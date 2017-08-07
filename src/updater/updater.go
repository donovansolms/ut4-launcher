package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/sethgrid/pester"
)

// Config is the configuration for the eupdater
type Config struct {
	InstallsDir string
	WorkingDir  string
	UpdateURL   string
	ClientID    string
	SendStats   bool
}

// Updater implements the updating logic for Unreal Tournament installations
type Updater struct {
	config      Config
	versionMaps VersionMaps
}

// New creates a new updater instance
func New(config Config) (*Updater, error) {
	// Everything we need it set?
	if config.InstallsDir == "" {
		return nil, errors.New("InstallsDir must be set")
	}
	if config.WorkingDir == "" {
		return nil, errors.New("WorkingDir must be set")
	}
	if config.UpdateURL == "" {
		return nil, errors.New("UpdateURL must be set")
	}
	if config.ClientID == "" {
		return nil, errors.New("ClientID must be set")
	}
	// Paths exist?
	fileInfo, err := os.Stat(config.InstallsDir)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("InstallDir must be a directory")
	}
	fileInfo, err = os.Stat(config.WorkingDir)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("InstallDir must be a directory")
	}
	// All ok
	updater := Updater{
		config: config,
	}
	err = updater.updateVersionMap()
	if err != nil {
		return nil, fmt.Errorf("Unable to update version map: %s", err.Error())
	}
	return &updater, nil
}

// DownloadUpdate downloads the update given by IsUpdateAvailable and
// returns true if downloaded successfully
// Provides a feedback and cancel channels to provide progress to the UI
func (updater *Updater) DownloadUpdate(
	packageURL string,
	savePath string,
	cancelChan chan bool,
	feedbackChan chan DownloadProgressEvent) (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	client := grab.NewClient()
	req, err := grab.NewRequest(savePath, packageURL)
	if err != nil {
		return false, err
	}
	req.WithContext(ctx)

	resp := client.Do(req)
	if resp.HTTPResponse.StatusCode >= 300 {
		return false,
			fmt.Errorf("Received non-2XX status code: %s", resp.HTTPResponse.Status)
	}

	t := time.NewTicker(time.Second)
	defer t.Stop()

UpdateLoop:
	for {
		select {
		case <-t.C:
			// On every tick, send an update
			feedbackChan <- DownloadProgressEvent{
				Filename:  resp.Filename,
				Mbps:      resp.BytesPerSecond() / 1024.00 / 1024.00,
				ETA:       float64(resp.ETA().Second()),
				Completed: false,
				Percent:   resp.Progress() * 100.00,
			}
		case <-resp.Done:
			feedbackChan <- DownloadProgressEvent{
				Filename:  resp.Filename,
				Mbps:      resp.BytesPerSecond() / 1024.00 / 1024.00,
				ETA:       float64(resp.ETA().Second()),
				Completed: true,
				Percent:   resp.Progress() * 100.00,
			}
			break UpdateLoop
		case <-cancelChan:
			cancel()
			break UpdateLoop
		}
	}
	if err := resp.Err(); err != nil {
		return false, err
	}
	return true, nil
}

// IsUpdateAvailable checks if a new version is available, returns true
// with the new version if available
func (updater *Updater) IsUpdateAvailable() (bool, string, string, error) {
	latestVersion, err := updater.GetLatestVersion()
	if err != nil {
		return false, "", "", err
	}
	osDistribution := OSDistribution{
		Distribution:           "Optout",
		DistributionID:         "optout",
		DistributionPrettyName: "Optout",
		KernelVersion:          "Linux Optout",
		DistributionVersion:    "0.0",
	}
	var versions []string
	if updater.config.SendStats {
		osDistribution = updater.GetOSDistribution()
		installedVersions, err := updater.GetVersionList()
		if err == nil {
			for _, version := range installedVersions {
				versions = append(versions, version.Version)
			}
		}
	}
	updateCheckRequest := UpdateCheckRequest{
		ClientID:       updater.config.ClientID,
		OS:             osDistribution,
		Versions:       versions,
		CurrentVersion: latestVersion.Version,
	}
	checkJSON, err := json.Marshal(updateCheckRequest)
	if err != nil {
		return false, "", "", err
	}

	client := pester.New()
	client.Concurrency = 1
	client.MaxRetries = 1
	client.Backoff = pester.DefaultBackoff
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/%s/%s", updater.config.UpdateURL, "ut4", "check"),
		bytes.NewReader(checkJSON))
	if err != nil {
		return false, "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", "", err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(data))
	os.Exit(0)
	var response UpdateCheckResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return false, "", "", err
	}
	return response.UpdateAvailable,
		response.LatestVersion,
		response.UpdateURL,
		nil
}

// cloneLatestVersionTo copies the latest version to a new version folder
// and returns the new base path of the installation
func (updater *Updater) cloneLatestVersionTo(
	version string,
	overwrite bool) (string, error) {
	newInstallPath, err := updater.GetVersionPath(version, overwrite)
	if err != nil {
		err = os.RemoveAll(newInstallPath)
		if err != nil {
			// Probably permission error
			return "", err
		}
		err = os.MkdirAll(newInstallPath, 0755)
		if err != nil {
			// Probably permission error
			return "", err
		}
	}
	latestVersion, err := updater.GetLatestVersion()
	if err != nil {
		// No installed version?
		return "", err
	}
	err = CopyDir(latestVersion.Path, newInstallPath)
	if err != nil {
		return "", err
	}
	return newInstallPath, nil
}

// GetLatestVersion returns the latest version installed
func (updater *Updater) GetLatestVersion() (Version, error) {
	versions, err := updater.GetVersionList()
	if err != nil {
		return Version{}, err
	}
	if len(versions) == 0 {
		return Version{}, errors.New("No Unreal Tournament versions installed")
	}
	return versions[0], nil
}

// GetVersionList returns the available installed versions as [version][path]
func (updater *Updater) GetVersionList() ([]Version, error) {
	fileInfo, err := os.Stat(updater.config.InstallsDir)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() == false {
		return nil, errors.New("The install path must be a directory")
	}

	files, err := ioutil.ReadDir(updater.config.InstallsDir)
	if err != nil {
		return nil, err
	}

	var versions []Version
	for _, file := range files {
		if file.IsDir() {
			versionPath, err := updater.GetVersionPath(file.Name(), false)
			if err != nil {
				continue
			}
			version := Version{
				Path: versionPath,
				VersionMap: updater.versionMaps.GetVersionMapByVersionNumber(
					file.Name()),
			}
			versions = append(versions, version)
		}
	}
	// Reverse the order so that the latest is at the top
	sort.Sort(ByVersion(versions))
	return versions, nil
}

// GetVersionPath returns the path to the version, setting mustNotExist to true
// will return an error if the path exists
func (updater *Updater) GetVersionPath(
	version string, mustNotExist bool) (string, error) {
	newInstallPath := filepath.Join(updater.config.InstallsDir, version)
	fileInfo, err := os.Stat(newInstallPath)
	if err == nil && fileInfo.IsDir() && mustNotExist {
		return newInstallPath, fmt.Errorf("The update path '%s' already exists", newInstallPath)
	}
	return newInstallPath, nil
}

// GetOSDistribution retrieves the kernel and distribution versions
func (updater *Updater) GetOSDistribution() OSDistribution {
	var osDistribution OSDistribution

	// /etc/os-release is the preferred way to check for distribution,
	// if it exists, we'll use it, otherwise just check for another *-release
	// file and use a part of it. This isn't critical to the updater.
	hasReleaseFile := true
	releaseBytes, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		// File doesn't exist, check if the next one does
		releaseBytes, err = ioutil.ReadFile("/usr/lib/os-release")
		if err != nil {
			// still no release file
			hasReleaseFile = false
		}
	}
	releaseContents := make(map[string]string)
	if hasReleaseFile {
		for _, line := range strings.Split(string(releaseBytes), "\n") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.Replace(strings.TrimSpace(parts[1]), "\"", "", -1)
				releaseContents[key] = value
			}
		}
	} else {
		_ = filepath.Walk("/etc",
			func(path string, f os.FileInfo, _ error) error {
				if !f.IsDir() {
					r, walkErr := regexp.MatchString("release", f.Name())
					if walkErr == nil && r {
						// This is a lazy way since this is not really important
						parts := strings.Split(f.Name(), "-")
						if len(parts) == 2 {
							releaseContents["ID"] = strings.Title(parts[0])
							releaseContents["NAME"] = releaseContents["ID"] + " Linux"
							releaseContents["PRETTY_NAME"] = releaseContents["NAME"]
						}
					}
				}
				return nil
			})
		if len(releaseContents) == 0 {
			releaseContents["ID"] = "Generic"
			releaseContents["NAME"] = "Generic Linux"
			releaseContents["PRETTY_NAME"] = "Generic Linux"
		}
	}

	if _, ok := releaseContents["NAME"]; ok {
		osDistribution.Distribution = releaseContents["NAME"]
	}
	if _, ok := releaseContents["ID"]; ok {
		osDistribution.DistributionID = releaseContents["ID"]
	}
	if _, ok := releaseContents["VERSION_ID"]; ok {
		osDistribution.DistributionVersion = releaseContents["VERSION_ID"]
	}
	if _, ok := releaseContents["PRETTY_NAME"]; ok {
		osDistribution.DistributionPrettyName = releaseContents["PRETTY_NAME"]
	}

	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		// Could not execute uname -r
		osDistribution.KernelVersion = "Unknown"
	} else {
		rawVersion := string(out)
		parts := strings.Split(rawVersion, "-")
		if len(parts) > 0 {
			osDistribution.KernelVersion = parts[0]
		} else {
			osDistribution.KernelVersion = rawVersion
		}
	}
	osDistribution.HasElectron = updater.hasElectron()
	return osDistribution
}

func (updater *Updater) hasElectron() bool {
	_, err := exec.Command("which", "electron").Output()
	if err == nil {
		return true
	}
	return false
}

// updateVersionMap retrieves the version map from the update server
// and saves a copy locally
func (updater *Updater) updateVersionMap() error {
	versionMapURL := fmt.Sprintf("%s/%s/%s",
		updater.config.UpdateURL,
		"ut4",
		"versionmap")
	var mapReader io.ReadCloser
	response, err := http.Get(versionMapURL)
	if err != nil {
		// We were unable to fetch the version map from the remote server
		// now we can check if a local copy exists
		// Declaring localErr to avoid shadowing mapReader
		var localErr error
		mapReader, localErr = os.Open(filepath.Join(
			updater.config.InstallsDir,
			"versionmap.json"))
		if localErr != nil {
			return fmt.Errorf("Remote returned '%s' and local copy returned '%s'",
				err.Error(),
				localErr.Error())
		}
	} else {
		// Response received
		mapReader = response.Body
	}

	versionMapBytes, err := ioutil.ReadAll(mapReader)
	if err != nil {
		return err
	}

	var versionMaps VersionMaps
	err = json.Unmarshal(versionMapBytes, &versionMaps)
	if err != nil {
		return err
	}
	defer mapReader.Close()
	updater.versionMaps = versionMaps

	// Write a local cache for the versionmap
	err = ioutil.WriteFile(filepath.Join(
		updater.config.InstallsDir,
		"versionmap.json"), versionMapBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

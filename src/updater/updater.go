package updater

import (
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

// IsUpdateAvailable checks if a new version is available, returns true
// with the new version if available
func (updater *Updater) IsUpdateAvailable() (bool, string, error) {

	return false, "", nil
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

	return osDistribution
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

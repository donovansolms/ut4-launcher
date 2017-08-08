package launcher

import (
	"github.com/donovansolms/ut4-launcher/src/updater"
	log "github.com/sirupsen/logrus"
)

// Config contains the configuration for the launcher
type Config struct {
	InstallsDir     string
	WorkingDir      string
	UpdateURL       string
	UpdateSendStats bool
	LogPath         string
	LogLevel        string
	ServerPort      int
}

// Launcher contains the main runnable logic for the UT4 loader
type Launcher struct {
	config Config
}

// New creates a new instance of the launcher from the given config
func New(config Config, isFirstRun bool) (*Launcher, error) {

	if !isFirstRun {
		updaterConfig := updater.Config{
			InstallsDir: config.InstallsDir,
			WorkingDir:  config.WorkingDir,
			UpdateURL:   config.UpdateURL,
			SendStats:   true,
			ClientID:    "001",
		}
		updater, err := updater.New(updaterConfig)
		if err != nil {
			return nil, err
		}
		_ = updater

		//fmt.Println(updater.GetVersionList())
		//fmt.Println(updater.GetLatestVersion())
		//fmt.Println(updater.GetOSDistribution())
		//fmt.Println(updater.IsUpdateAvailable())
		//fmt.Println(updater.GetOSDistribution().HasElectron)
		//
	}
	launcher := Launcher{
		config: config,
	}

	return &launcher, nil
}

// Launch starts the launcher
func (launcher *Launcher) Launch() error {
	log.Info("Starting launcher")
	return nil
}

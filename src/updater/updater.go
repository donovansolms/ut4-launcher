package updater

import (
	"errors"
	"os"
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
	config Config
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
	return &updater, nil
}

// IsUpdateAvailable checks if a new version is available, returns true
// with the new version if available
func (updater *Updater) IsUpdateAvailable() (bool, string, error) {

	return false, "", nil
}

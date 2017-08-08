// Package main implements the main main runnable UT4 launcher
package main

import (
	"os"

	"github.com/donovansolms/ut4-launcher/src/launcher"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	// TODO: https://github.com/sirupsen/logrus/issues/156
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "Jan 02 15:04:05",
	})

	firstRun := false
	viper.SetConfigName("ut4-launcher")
	viper.AddConfigPath("$HOME/.ut4-launcher")
	viper.AddConfigPath("/opt/ut4-launcher/")
	viper.AddConfigPath("/usr/local/ut4-launcher/")
	viper.AddConfigPath("/etc/opt/ut4-launcher/")
	viper.AddConfigPath("/etc/ut4-launcher/")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("Unable to read config file: %s", err.Error())
		log.Debug("No config file available, probably first run")
		log.Info("Start setup")
		// Config file doesn't exist, we need to run the initial
		// first run setup
		firstRun = true
	}

	config := launcher.Config{
		InstallsDir:     viper.GetString("InstallsDir"),
		WorkingDir:      viper.GetString("WorkingDir"),
		UpdateURL:       viper.GetString("Updater.URL"),
		UpdateSendStats: viper.GetBool("Updater.SendStats"),
		LogPath:         viper.GetString("Log"),
		LogLevel:        viper.GetString("LogLevel"),
		ServerPort:      viper.GetInt("Server.Port"),
	}

	if firstRun {
		// For first run, set the defaults
		config = launcher.Config{
			InstallsDir:     "",
			WorkingDir:      "",
			UpdateURL:       "http://update.donovansolms.local",
			UpdateSendStats: true,
			LogPath:         "/var/log/ut4-updater/debug.log",
			LogLevel:        "DEBUG",
			ServerPort:      5000,
		}
	}
	logLevelConst, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Panicf("Invalid log level: %s", err.Error())
	}
	log.SetLevel(logLevelConst)

	launcher, err := launcher.New(config, firstRun)
	if err != nil {
		log.Panicf("Can't create launcher: %s", err.Error())
	}

	err = launcher.Launch()
	if err != nil {
		log.Panicf("Launch failed: %s", err.Error())
	}
}

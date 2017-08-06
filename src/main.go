// Package main implements the main main runnable UT4 launcher
package main

import "github.com/donovansolms/ut4-launcher/src/updater"

func main() {
	// TODO: Set up logging
	// https://github.com/sirupsen/logrus/issues/156
	//
	// TODO: Load config file
	updaterConfig := updater.Config{
		InstallsDir: "/tmp",
		WorkingDir:  "/tmp",
		UpdateURL:   "http://update.donovansolms.local",
		SendStats:   true,
		ClientID:    "001",
	}
	updater, err := updater.New(updaterConfig)
	if err != nil {
		panic(err)
	}
	_ = updater
}

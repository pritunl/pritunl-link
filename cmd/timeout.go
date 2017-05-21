package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func DisconnectedTimeoutOn() (err error) {
	config.Config.DisableDisconnectedRestart = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.timeout: Disconnected timeout enabled")

	return
}

func DisconnectedTimeoutOff() (err error) {
	config.Config.DisableDisconnectedRestart = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.timeout: Disconnected timeout disabled")

	return
}

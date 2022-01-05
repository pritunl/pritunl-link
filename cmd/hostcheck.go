package cmd

import (
	"github.com/pritunl/pritunl-link/config"
	"github.com/sirupsen/logrus"
)

func HostCheckOn() (err error) {
	config.Config.SkipHostCheck = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.hostcheck: Host checking enabled")

	return
}

func HostCheckOff() (err error) {
	config.Config.SkipHostCheck = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.hostcheck: Host checking disabled")

	return
}

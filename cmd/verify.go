package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func VerifyOn() (err error) {
	config.Config.SkipVerify = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.verify: Certificate verification enabled")

	return
}

func VerifyOff() (err error) {
	config.Config.SkipVerify = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.verify: Certificate verification disabled")

	return
}

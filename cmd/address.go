package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func Address(address string) (err error) {
	config.Config.PublicAddress = address

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"public_address": config.Config.PublicAddress,
	}).Info("cmd.address: Public address set")

	return
}

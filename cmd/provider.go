package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func Provider(provider string) (err error) {
	config.Config.Provider = provider

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"provider": config.Config.Provider,
	}).Info("cmd.provider: Set provider")

	return
}

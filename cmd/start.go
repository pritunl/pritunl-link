package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/constants"
)

func Start() {
	logrus.WithFields(logrus.Fields{
		"version": constants.Version,
	}).Info("cmd.app: Starting link")
}

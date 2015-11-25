package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/constants"
)

func Start() {
	opts := getOptions()

	logrus.WithFields(logrus.Fields{
		"id":      opts.Id,
		"host":    opts.Host,
		"version": constants.Version,
	}).Info("cmd.app: Starting app node")
}

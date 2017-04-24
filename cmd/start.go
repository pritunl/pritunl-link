package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/utils"
)

func Start() (err error) {
	logrus.WithFields(logrus.Fields{
		"version": constants.Version,
	}).Info("cmd.start: Starting link")

	err = utils.NetInit()
	if err != nil {
		return
	}

	return
}

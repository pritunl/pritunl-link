package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/sync"
	"time"
)

func Start() (err error) {
	sync.Init()

	logrus.WithFields(logrus.Fields{
		"version":        constants.Version,
		"local_address":  state.GetLocalAddress(),
		"public_address": state.GetPublicAddress(),
	}).Info("cmd.start: Starting link")

	for {
		time.Sleep(1 * time.Second)
	}

	return
}

package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func Remove(uri string) (err error) {
	for i, u := range config.Config.Uris {
		if uri == u {
			config.Config.Uris = append(
				config.Config.Uris[:i],
				config.Config.Uris[i+1:]...,
			)

			break
		}
	}

	logrus.WithFields(logrus.Fields{
		"uris": config.Config.Uris,
	}).Info("cmd.remove: Removed URI")

	return
}

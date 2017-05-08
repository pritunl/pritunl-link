package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func Add(uri string) (err error) {
	exists := false

	for _, u := range config.Config.Uris {
		if uri == u {
			exists = true
		}
	}

	if !exists {
		config.Config.Uris = append(config.Config.Uris, uri)

		err = config.Save()
		if err != nil {
			return
		}
	}

	logrus.WithFields(logrus.Fields{
		"uris": config.Config.Uris,
	}).Info("cmd.add: Added URI")

	return
}

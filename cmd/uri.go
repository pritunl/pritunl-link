package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func List() (err error) {
	logrus.WithFields(logrus.Fields{
		"uris": config.Config.Uris,
	}).Info("cmd.uri: List URI")

	return
}

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
	}).Info("cmd.uri: Added URI")

	return
}

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
	}).Info("cmd.uri: Removed URI")

	return
}

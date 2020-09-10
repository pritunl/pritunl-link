package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func Add(uri string) (err error) {
	exists := false

	if uri != "" {
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
	}

	logrus.WithFields(logrus.Fields{
		"uris": config.Config.Uris,
	}).Info("cmd.uri: Added URI")

	return
}

func Remove(uri string) (err error) {
	exists := false

	for i, u := range config.Config.Uris {
		if uri == u {
			exists = true

			config.Config.Uris = append(
				config.Config.Uris[:i],
				config.Config.Uris[i+1:]...,
			)

			break
		}
	}

	if exists {
		err = config.Save()
		if err != nil {
			return
		}
	}

	logrus.WithFields(logrus.Fields{
		"uris": config.Config.Uris,
	}).Info("cmd.uri: Removed URI")

	return
}

func Clear() (err error) {
	config.Config.Uris = []string{}

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"uris": config.Config.Uris,
	}).Info("cmd.clear: Cleared URI")

	return
}

func List() (err error) {
	for _, u := range config.Config.Uris {
		fmt.Println(u)
	}

	return
}

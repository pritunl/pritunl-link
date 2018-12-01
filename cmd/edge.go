package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func EdgeUsername(username string) (err error) {
	config.Config.Edge.Username = username

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"username": config.Config.Edge.Username,
	}).Info("cmd.edge: Set EdgeRouter username")

	return
}

func EdgePassword(password string) (err error) {
	config.Config.Edge.Password = password

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"password": config.Config.Edge.Password,
	}).Info("cmd.edge: Set EdgeRouter password")

	return
}

func EdgeHostname(hostname string) (err error) {
	config.Config.Edge.Hostname = hostname

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"hostname": config.Config.Edge.Hostname,
	}).Info("cmd.edge: Set EdgeRouter hostname")

	return
}

func EdgePortOn() (err error) {
	config.Config.Edge.DisablePort = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.edge: EdgeRouter port forwarding enabled")

	return
}

func EdgePortOff() (err error) {
	config.Config.Edge.DisablePort = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.edge: EdgeRouter port forwarding disabled")

	return
}

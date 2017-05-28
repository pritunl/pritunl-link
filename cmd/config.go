package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func LocalAddress(address string) (err error) {
	config.Config.LocalAddress = address

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"local_address": config.Config.LocalAddress,
	}).Info("cmd.address: Local address set")

	return
}

func PublicAddress(address string) (err error) {
	config.Config.PublicAddress = address

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"public_address": config.Config.PublicAddress,
	}).Info("cmd.address: Public address set")

	return
}

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

func DisconnectedTimeoutOn() (err error) {
	config.Config.DisableDisconnectedRestart = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.timeout: Disconnected timeout enabled")

	return
}

func DisconnectedTimeoutOff() (err error) {
	config.Config.DisableDisconnectedRestart = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.timeout: Disconnected timeout disabled")

	return
}

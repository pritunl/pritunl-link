package cmd

import (
	"strconv"

	"github.com/pritunl/pritunl-link/config"
	"github.com/sirupsen/logrus"
)

func HetznerToken(token string) (err error) {
	config.Config.Hetzner.Token = token

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"token": config.Config.Hetzner.Token,
	}).Info("cmd.hetzner: Set Hetzner token")

	return
}

func HetznerNetworkId(networkId string) (err error) {
	network, err := strconv.Atoi(networkId)
	if err != nil {
		return
	}

	config.Config.Hetzner.NetworkId = network

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"network_id": config.Config.Hetzner.NetworkId,
	}).Info("cmd.hetzner: Set Hetzner network id")

	return
}

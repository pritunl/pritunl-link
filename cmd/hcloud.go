package cmd

import (
	"strconv"

	"github.com/pritunl/pritunl-link/config"
	"github.com/sirupsen/logrus"
)

func HcloudToken(token string) (err error) {
	config.Config.Hcloud.Token = token

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"token": config.Config.Hcloud.Token,
	}).Info("cmd.hcloud: Set Hcloud token")

	return
}

func HcloudNetworkId(networkId string) (err error) {
	network, err := strconv.Atoi(networkId)
	if err != nil {
		return
	}

	config.Config.Hcloud.NetworkId = network

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"networkId": config.Config.Hcloud.NetworkId,
	}).Info("cmd.hcloud: Set Hcloud network id")

	return
}

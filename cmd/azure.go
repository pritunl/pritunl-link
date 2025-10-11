package cmd

import (
	"github.com/pritunl/pritunl-link/config"
	"github.com/sirupsen/logrus"
)

func AzureNetResourceGroup(val string) (err error) {
	config.Config.Azure.NetworkResourceGroup = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"region": config.Config.Azure.NetworkResourceGroup,
	}).Info("cmd.unifi: Set Azure virtual network resource group")

	return
}

package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func PritunlHostname(hostname string) (err error) {
	config.Config.Pritunl.Hostname = hostname

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"hostname": config.Config.Pritunl.Hostname,
	}).Info("cmd.pritunl: Set Pritunl hostname")

	return
}

func PritunlVpc(vpcId string) (err error) {
	config.Config.Pritunl.VpcId = vpcId

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"vpc_id": config.Config.Pritunl.VpcId,
	}).Info("cmd.pritunl: Set Pritunl VPC")

	return
}

func PritunlToken(token string) (err error) {
	config.Config.Pritunl.Token = token

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"token": config.Config.Pritunl.Token,
	}).Info("cmd.pritunl: Set Pritunl token")

	return
}

func PritunlSecret(secret string) (err error) {
	config.Config.Pritunl.Secret = secret

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"secret": config.Config.Pritunl.Secret,
	}).Info("cmd.pritunl: Set Pritunl secret")

	return
}

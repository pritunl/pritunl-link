package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/pritunl/pritunl-link/config"
)

func OracleRegion(val string) (err error) {
	config.Config.Oracle.Region = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"region": config.Config.Oracle.Region,
	}).Info("cmd.unifi: Set Oracle region")

	return
}

func OraclePrivateKey(val string) (err error) {
	config.Config.Oracle.PrivateKey = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"private_key": config.Config.Oracle.PrivateKey,
	}).Info("cmd.unifi: Set Oracle private key")

	return
}

func OracleUserOcid(val string) (err error) {
	config.Config.Oracle.UserOcid = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_ocid": config.Config.Oracle.UserOcid,
	}).Info("cmd.unifi: Set Oracle user OCID")

	return
}

func OracleTenancyOcid(val string) (err error) {
	config.Config.Oracle.TenancyOcid = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"tenancy_ocid": config.Config.Oracle.TenancyOcid,
	}).Info("cmd.unifi: Set Oracle tenancy OCID")

	return
}

func OracleCompartmentOcid(val string) (err error) {
	config.Config.Oracle.CompartmentOcid = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"compartment_ocid": config.Config.Oracle.CompartmentOcid,
	}).Info("cmd.unifi: Set Oracle compartment OCID")

	return
}

func OracleVncOcid(val string) (err error) {
	config.Config.Oracle.VncOcid = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"vnc_ocid": config.Config.Oracle.VncOcid,
	}).Info("cmd.unifi: Set Oracle vnc OCID")

	return
}

func OraclePrivateIpOcid(val string) (err error) {
	config.Config.Oracle.PrivateIpOcid = val

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"private_ip_ocid": config.Config.Oracle.PrivateIpOcid,
	}).Info("cmd.unifi: Set Oracle private IP OCID")

	return
}

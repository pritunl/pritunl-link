package cmd

import (
	"github.com/pritunl/pritunl-link/config"
	"github.com/sirupsen/logrus"
)

func DefaultInterface(iface string) (err error) {
	config.Config.DefaultInterface = iface

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"default_interface": config.Config.DefaultInterface,
	}).Info("cmd.config: Default interface set")

	return
}

func DefaultGateway(gateway string) (err error) {
	config.Config.DefaultGateway = gateway

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"default_gateway": config.Config.DefaultGateway,
	}).Info("cmd.config: Default gateway set")

	return
}

func LocalAddress(address string) (err error) {
	config.Config.LocalAddress = address

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"local_address": config.Config.LocalAddress,
	}).Info("cmd.config: Local address set")

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
	}).Info("cmd.config: Public address set")

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
	}).Info("cmd.config: Set provider")

	return
}

func DisconnectedTimeoutOn() (err error) {
	config.Config.DisableDisconnectedRestart = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Disconnected timeout enabled")

	return
}

func DisconnectedTimeoutOff() (err error) {
	config.Config.DisableDisconnectedRestart = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Disconnected timeout disabled")

	return
}

func AdvertiseUpdateOn() (err error) {
	config.Config.DisableAdvertiseUpdate = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Advertise update enabled")

	return
}

func AdvertiseUpdateOff() (err error) {
	config.Config.DisableAdvertiseUpdate = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Advertise update disabled")

	return
}

func RemoveRoutesOn() (err error) {
	config.Config.DeleteRoutes = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Remove routes enabled")

	return
}

func RemoveRoutesOff() (err error) {
	config.Config.DeleteRoutes = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Remove routes disabled")

	return
}

func DirectSshOn() (err error) {
	config.Config.DirectSsh = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Direct SSH enabled")

	return
}

func DirectSshOff() (err error) {
	config.Config.DirectSsh = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Direct SSH disabled")

	return
}

func FirewallOn() (err error) {
	config.Config.Firewall = true

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Firewall enabled")

	return
}

func FirewallOff() (err error) {
	config.Config.Firewall = false

	err = config.Save()
	if err != nil {
		return
	}

	logrus.Info("cmd.config: Firewall disabled")

	return
}

func AddCustomOption(opt string) (err error) {
	config.Config.CustomOptions = append(config.Config.CustomOptions, opt)

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"custom_options": config.Config.CustomOptions,
	}).Info("cmd.config: Added custom option")

	return
}

func ClearCustomOptions() (err error) {
	config.Config.CustomOptions = []string{}

	err = config.Save()
	if err != nil {
		return
	}

	logrus.WithFields(logrus.Fields{
		"custom_options": config.Config.CustomOptions,
	}).Info("cmd.config: Cleared custom options")

	return
}

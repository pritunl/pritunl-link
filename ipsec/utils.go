package ipsec

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/config"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/utils"
	"net"
	"strings"
)

func GetDirectSubnet() (network *net.IPNet, err error) {
	networkStr := config.Config.DirectSubnet
	if networkStr == "" {
		networkStr = defaultDirectNetwork
	}

	_, network, err = net.ParseCIDR(networkStr)
	if err != nil {
		err = errortypes.ParseError{
			errors.Wrap(err, "ipsec: Failed to prase direct subnet"),
		}
		return
	}

	return
}

func GetDirectCidr() string {
	networkStr := config.Config.DirectSubnet
	if networkStr == "" {
		networkStr = defaultDirectNetwork
	}

	networkSpl := strings.Split(networkStr, "/")

	return networkSpl[len(networkSpl)-1]
}

func GetDirectServerIp() (ip net.IP, err error) {
	network, err := GetDirectSubnet()
	if err != nil {
		return
	}

	ip = network.IP
	utils.IncIpAddress(ip)

	return
}

func GetDirectClientIp() (ip net.IP, err error) {
	network, err := GetDirectSubnet()
	if err != nil {
		return
	}

	ip = network.IP
	utils.IncIpAddress(ip)
	utils.IncIpAddress(ip)

	return
}

func GetDirectMode() (mode string) {
	mode = config.Config.DirectMode
	if mode == "" {
		mode = defaultDirectMode
	}
	return
}

package ipsec

import (
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/constants"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
)

var (
	tunnelLocal  = ""
	tunnelRemote = ""
)

func StartTunnel(stat *state.State) (err error) {
	if GetDirectMode() != DirectGre {
		StopTunnel()
		return
	}

	peerLocal := ""
	if len(stat.Links) > 0 && len(stat.Links[0].RightSubnets) > 0 {
		peerLocal = stat.Links[0].RightSubnets[0]
		peerLocal = strings.SplitN(peerLocal, "/", 2)[0]
	}

	if peerLocal == "" {
		err = &errortypes.ReadError{
			errors.New("ipsec: Missing peer local address"),
		}
		return
	}

	newTunnelLocal := state.GetLocalAddress()
	newTunnelRemote := peerLocal

	if newTunnelLocal == tunnelLocal && newTunnelRemote == tunnelRemote {
		return
	}
	StopTunnel()

	if newTunnelLocal == "" || newTunnelRemote == "" {
		return
	}

	tunnelLocal = newTunnelLocal
	tunnelRemote = newTunnelRemote

	logrus.WithFields(logrus.Fields{
		"local":  newTunnelLocal,
		"remote": newTunnelRemote,
	}).Info("ipsec: Starting GRE tunnel")

	err = utils.Exec("",
		"ip", "tunnel",
		"add", DirectIface,
		"mode", "gre",
		"local", newTunnelLocal,
		"remote", newTunnelRemote,
	)
	if err != nil {
		return
	}

	err = utils.Exec("",
		"ip", "link",
		"set", DirectIface, "up",
	)
	if err != nil {
		return
	}

	var directAddrIp net.IP
	if stat.Type == state.DirectClient {
		directAddrIp, err = GetDirectClientIp()
	} else {
		directAddrIp, err = GetDirectServerIp()
	}
	if err != nil {
		return
	}
	directAddr := directAddrIp.String()

	err = utils.Exec("",
		"ip", "addr",
		"add", directAddr+"/"+GetDirectCidr(),
		"dev", DirectIface,
	)
	if err != nil {
		return
	}

	return
}

func StopTunnel() {
	if tunnelLocal != "" && tunnelRemote != "" {
		logrus.WithFields(logrus.Fields{
			"local":  tunnelLocal,
			"remote": tunnelRemote,
		}).Info("ipsec: Stopping GRE tunnel")
	}

	utils.ExecSilent("",
		"ip", "tunnel",
		"del", DirectIface,
	)
	tunnelLocal = ""
	tunnelRemote = ""
}

func StopWg() {
	curWgIfaces, activeWgIfaces, err := GetWgIfaces()
	if err != nil {
		return
	}

	for ifaceInf := range curWgIfaces.Iter() {
		iface := ifaceInf.(string)
		confPth := path.Join(constants.WgDirPath,
			fmt.Sprintf("%s.conf", iface))

		err = utils.Exec("", "wg-quick", "down", iface)
		if err != nil {
			if activeWgIfaces.Contains(iface) {
				logrus.WithFields(logrus.Fields{
					"wg_iface": iface,
					"error":    err,
				}).Error("state: Error bringing down wg conf")
			}
			err = nil
		}

		os.Remove(confPth)
	}
}

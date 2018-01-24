package ipsec

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
	"github.com/pritunl/pritunl-link/state"
	"github.com/pritunl/pritunl-link/utils"
	"net"
	"strings"
)

func StartTunnel(stat *state.State) (err error) {
	peerLocal := stat.Links[0].RightSubnets[0]
	peerLocal = strings.SplitN(peerLocal, "/", 2)[0]

	if peerLocal == "" {
		err = &errortypes.ReadError{
			errors.New("ipsec: Missing peer local address"),
		}
		return
	}

	err = utils.Exec("",
		"ip", "tunnel",
		"add", DirectIface,
		"mode", "gre",
		"local", state.GetLocalAddress(),
		"remote", peerLocal,
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
	utils.ExecSilent("",
		"ip", "tunnel",
		"del", DirectIface,
	)
}

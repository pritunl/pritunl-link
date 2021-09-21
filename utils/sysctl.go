package utils

import (
	"fmt"
	"net"
	"strings"

	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-link/errortypes"
)

func NetInit() (err error) {
	err = ExecSilent(
		"", "sysctl", "-w", "net.ipv4.ip_forward=1")
	if err != nil {
		return
	}

	err = ExecSilent(
		"", "sysctl", "-w", "net.ipv4.conf.all.send_redirects=0")
	if err != nil {
		return
	}

	err = ExecSilent(
		"", "sysctl", "-w", "net.ipv4.conf.default.send_redirects=0")
	if err != nil {
		return
	}

	err = ExecSilent(
		"", "sysctl", "-w", "net.ipv4.conf.all.accept_redirects=0")
	if err != nil {
		return
	}

	err = ExecSilent(
		"", "sysctl", "-w", "net.ipv4.conf.default.accept_redirects=0")
	if err != nil {
		return
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "utils: Failed to read network interfaces"),
		}
		return
	}

	ExecSilent("", "sysctl", "-w", "net.ipv6.conf.all.accept_ra=2")
	ExecSilent("", "sysctl", "-w", "net.ipv6.conf.default.accept_ra=2")
	ExecSilent("", "sysctl", "-w", "net.ipv6.conf.all.forwarding=1")
	ExecSilent("", "sysctl", "-w", "net.ipv6.conf.default.forwarding=1")

	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "lo") {
			continue
		}

		ExecSilent("", "sysctl", "-w",
			fmt.Sprintf("net.ipv6.conf.%s.accept_ra=2", iface.Name))
	}

	return
}

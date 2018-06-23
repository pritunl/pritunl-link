package utils

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

	ExecSilent("", "sysctl", "-w", "net.ipv6.conf.all.forwarding=1")
	ExecSilent("", "sysctl", "-w", "net.ipv6.conf.default.forwarding=1")

	return
}

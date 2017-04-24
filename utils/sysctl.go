package utils

func NetInit() (err error) {
	err = Exec("", "sysctl", "-w", "net.ipv4.ip_forward=1")
	if err != nil {
		return
	}

	err = Exec("", "sysctl", "-w", "net.ipv4.conf.all.send_redirects=0")
	if err != nil {
		return
	}

	err = Exec("", "sysctl", "-w", "net.ipv4.conf.default.send_redirects=0")
	if err != nil {
		return
	}

	err = Exec("", "sysctl", "-w", "net.ipv4.conf.all.accept_redirects=0")
	if err != nil {
		return
	}

	err = Exec("", "sysctl", "-w", "net.ipv4.conf.default.accept_redirects=0")
	if err != nil {
		return
	}

	return
}

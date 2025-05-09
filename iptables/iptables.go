package iptables

import (
	"strings"

	"github.com/pritunl/pritunl-link/utils"
)

func clearIpTables(match string, ipv6 bool) (err error) {
	var iptablesExec string
	if ipv6 {
		iptablesExec = "ip6tables"
	} else {
		iptablesExec = "iptables"
	}

	output, err := utils.ExecOutput("", iptablesExec+"-save")
	if err != nil {
		return
	}

	table := ""
	outputLines := strings.Split(output, "\n")
	for _, line := range outputLines {
		if strings.HasPrefix(line, "*mangle") {
			table = "mangle"
			continue
		} else if strings.HasPrefix(line, "*nat") {
			table = "nat"
			continue
		} else if strings.HasPrefix(line, "*filter") {
			table = "filter"
			continue
		}

		if !strings.Contains(line, match) {
			continue
		}

		args := []string{
			"-t", table, "-D",
		}
		fields := strings.Fields(line)[1:]
		args = append(args, fields...)

		utils.Exec("", iptablesExec, args...)
	}

	return
}

func ClearIpTables() (err error) {
	return clearIpTables("--comment pritunl-link-direct", false)
}

func UpsertRule(table string, rule ...string) (err error) {
	args := []string{"-t", table, "-C"}
	args = append(args, rule...)

	e := utils.ExecSilent("", "iptables", args...)
	if e != nil {
		args = []string{"-t", table, "-A"}
		args = append(args, rule...)

		err = utils.Exec("", "iptables", args...)
		if err != nil {
			return
		}
	}

	return
}

func AllowPort(source, port, proto string) (err error) {
	var iptablesExec string
	if strings.Contains(source, ":") {
		iptablesExec = "ip6tables"
	} else {
		iptablesExec = "iptables"
	}

	rule := []string{
		"INPUT", "1",
		"-p", proto,
		"-m", proto,
		"--dport", port,
		"-s", source,
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-accept",
	}

	args := []string{"-C"}
	args = append(args, rule...)

	e := utils.ExecSilent("", iptablesExec, args...)
	if e != nil {
		args = []string{"-I"}
		args = append(args, rule...)

		err = utils.Exec("", iptablesExec, args...)
		if err != nil {
			return
		}
	}

	return
}

func DisallowPort(source, port, proto string) (err error) {
	var iptablesExec string
	if strings.Contains(source, ":") {
		iptablesExec = "ip6tables"
	} else {
		iptablesExec = "iptables"
	}

	rule := []string{
		"INPUT",
		"-p", proto,
		"-m", proto,
		"--dport", port,
		"-s", source,
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-accept",
	}

	args := []string{"-C"}
	args = append(args, rule...)

	e := utils.ExecSilent("", iptablesExec, args...)
	if e == nil {
		args = []string{"-D"}
		args = append(args, rule...)

		err = utils.Exec("", iptablesExec, args...)
		if err != nil {
			return
		}
	}

	return
}

func DropPort(port, proto string) (err error) {
	rule := []string{
		"INPUT",
		"-p", proto,
		"-m", proto,
		"--dport", port,
		"-j", "DROP",
		"-m", "comment",
		"--comment", "pritunl-link-drop",
	}

	for _, iptablesExec := range []string{"iptables", "ip6tables"} {
		args := []string{"-C"}
		args = append(args, rule...)

		e := utils.ExecSilent("", iptablesExec, args...)
		if e != nil {
			args = []string{"-A"}
			args = append(args, rule...)

			err = utils.Exec("", iptablesExec, args...)
			if err != nil {
				return
			}
		}
	}

	return
}

func AllowPortSet(source, portSet, proto string) (err error) {
	var iptablesExec string
	if strings.Contains(source, ":") {
		iptablesExec = "ip6tables"
	} else {
		iptablesExec = "iptables"
	}

	rule := []string{
		"INPUT", "1",
		"-p", proto,
		"-m", "set",
		"--match-set", portSet, "dst",
		"-s", source,
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-accept",
	}

	args := []string{"-C"}
	args = append(args, rule...)

	e := utils.ExecSilent("", iptablesExec, args...)
	if e != nil {
		args = []string{"-I"}
		args = append(args, rule...)

		err = utils.Exec("", iptablesExec, args...)
		if err != nil {
			return
		}
	}

	return
}

func DisallowPortSet(source, portSet, proto string) (err error) {
	var iptablesExec string
	if strings.Contains(source, ":") {
		iptablesExec = "ip6tables"
	} else {
		iptablesExec = "iptables"
	}

	rule := []string{
		"INPUT",
		"-p", proto,
		"-m", "set",
		"--match-set", portSet, "dst",
		"-s", source,
		"-j", "ACCEPT",
		"-m", "comment",
		"--comment", "pritunl-link-accept",
	}

	args := []string{"-C"}
	args = append(args, rule...)

	e := utils.ExecSilent("", iptablesExec, args...)
	if e == nil {
		args = []string{"-D"}
		args = append(args, rule...)

		err = utils.Exec("", iptablesExec, args...)
		if err != nil {
			return
		}
	}

	return
}

func DropPortSet(portSet, proto string) (err error) {
	rule := []string{
		"INPUT",
		"-p", proto,
		"-m", "set",
		"--match-set", portSet, "dst",
		"-j", "DROP",
		"-m", "comment",
		"--comment", "pritunl-link-drop",
	}

	for _, iptablesExec := range []string{"iptables", "ip6tables"} {
		args := []string{"-C"}
		args = append(args, rule...)

		e := utils.ExecSilent("", iptablesExec, args...)
		if e != nil {
			args = []string{"-A"}
			args = append(args, rule...)

			err = utils.Exec("", iptablesExec, args...)
			if err != nil {
				return
			}
		}
	}

	return
}

func InitWgIpset() (err error) {
	output, err := utils.ExecOutput("", "ipset", "list", "-n")
	if err != nil {
		return
	}

	for _, item := range strings.Split(output, "\n") {
		if strings.TrimSpace(item) == "wgp" {
			return
		}
	}

	err = utils.Exec("", "ipset", "create", "wgp",
		"bitmap:port", "range", "0-65535")
	if err != nil {
		return
	}

	return
}

func InitAcceptIpTables() (err error) {
	err = InitWgIpset()
	if err != nil {
		return
	}
	err = DropPort("500", "udp")
	if err != nil {
		return
	}
	err = DropPort("4500", "udp")
	if err != nil {
		return
	}
	err = DropPort("9790", "tcp")
	if err != nil {
		return
	}
	err = DropPortSet("wgp", "udp")
	if err != nil {
		return
	}

	return
}

func ClearWgIpset() (err error) {
	err = utils.Exec("", "ipset", "flush", "wgp")
	if err != nil {
		return
	}

	return
}

func RemoveWgIpset() (err error) {
	err = utils.Exec("", "ipset", "destroy", "wgp")
	if err != nil {
		return
	}

	return
}

func ClearAcceptIpTables() (err error) {
	err = clearIpTables("--comment pritunl-link-accept", false)
	if err != nil {
		return
	}

	err = clearIpTables("--comment pritunl-link-accept", true)
	if err != nil {
		return
	}

	return
}

func ClearDropIpTables() (err error) {
	err = clearIpTables("--comment pritunl-link-drop", false)
	if err != nil {
		return
	}

	err = clearIpTables("--comment pritunl-link-drop", true)
	if err != nil {
		return
	}

	return
}

func DeleteRule(table string, rule ...string) {
	args := []string{"-t", table, "-D"}
	args = append(args, rule...)
	utils.ExecSilent("", "iptables", args...)
}

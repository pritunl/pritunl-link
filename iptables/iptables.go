package iptables

import (
	"github.com/pritunl/pritunl-link/utils"
	"strings"
)

func ClearIpTables() (err error) {
	output, err := utils.ExecOutput("", "iptables-save")
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

		if !strings.Contains(line, "--comment pritunl-zero") {
			continue
		}

		args := []string{
			"-t", table, "-D",
		}
		fields := strings.Fields(line)[1:]
		args = append(args, fields...)

		utils.Exec("", "iptables", args...)
	}

	return
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

func DeleteRule(table string, rule ...string) {
	args := []string{"-t", table, "-D"}
	args = append(args, rule...)
	utils.ExecSilent("", "iptables", args...)
}

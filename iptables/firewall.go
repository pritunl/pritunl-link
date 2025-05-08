package iptables

import (
	"strconv"
	"sync"

	"github.com/dropbox/godropbox/container/set"
	"github.com/pritunl/pritunl-link/utils"
)

var (
	initialize   = true
	curHosts     = []string{}
	curWgPorts   = set.NewSet()
	iptablesLock = sync.Mutex{}
)

func SetHosts(hosts []string, ports []int) (err error) {
	iptablesLock.Lock()
	defer iptablesLock.Unlock()

	newHostsSet := set.NewSet()
	for _, host := range hosts {
		if host == "" {
			continue
		}
		newHostsSet.Add(host)
	}

	curHostsSet := set.NewSet()
	for _, host := range curHosts {
		curHostsSet.Add(host)
	}

	removeHosts := curHostsSet.Copy()
	removeHosts.Subtract(newHostsSet)

	addHosts := newHostsSet.Copy()
	addHosts.Subtract(curHostsSet)

	if removeHosts.Len() == 0 && addHosts.Len() == 0 {
		return
	}

	if initialize {
		ClearAcceptIpTables()
		initialize = false
	}

	InitWgIpset()

	for hostInf := range removeHosts.Iter() {
		host := hostInf.(string)

		err = DisallowPort(host, "500", "udp")
		if err != nil {
			return
		}
		err = DisallowPort(host, "4500", "udp")
		if err != nil {
			return
		}
		err = DisallowPort(host, "9790", "tcp")
		if err != nil {
			return
		}
		err = DisallowPortSet(host, "wgp", "udp")
		if err != nil {
			return
		}
	}

	for hostInf := range addHosts.Iter() {
		host := hostInf.(string)

		err = AllowPort(host, "500", "udp")
		if err != nil {
			return
		}
		err = AllowPort(host, "4500", "udp")
		if err != nil {
			return
		}
		err = AllowPort(host, "9790", "tcp")
		if err != nil {
			return
		}
		err = AllowPortSet(host, "wgp", "udp")
		if err != nil {
			return
		}
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

	curHosts = hosts

	curPorts := curWgPorts.Copy()

	newPorts := set.NewSet()
	for _, port := range ports {
		newPorts.Add(port)
	}

	addPorts := newPorts.Copy()
	addPorts.Subtract(curPorts)

	delPorts := curPorts.Copy()
	delPorts.Subtract(addPorts)

	for addPortInf := range addPorts.Iter() {
		addPort := addPortInf.(int)

		err = utils.Exec("", "ipset", "add", "wgp", strconv.Itoa(addPort))
		if err != nil {
			return
		}
	}

	for delPortInf := range delPorts.Iter() {
		delPort := delPortInf.(int)

		err = utils.Exec("", "ipset", "del", "wgp", strconv.Itoa(delPort))
		if err != nil {
			return
		}
	}

	curPorts = newPorts

	return
}

func ResetFirewall() {
	iptablesLock.Lock()
	defer iptablesLock.Unlock()

	initialize = true
	curHosts = []string{}
}

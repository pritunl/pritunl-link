package iptables

import (
	"sync"

	"github.com/dropbox/godropbox/container/set"
)

var (
	curHosts     = []string{}
	iptablesLock = sync.Mutex{}
)

func SetHosts(hosts []string) (err error) {
	iptablesLock.Lock()
	defer iptablesLock.Unlock()

	newHostsSet := set.NewSet()
	for _, host := range hosts {
		newHostsSet.Add(host)
	}

	curHostsSet := set.NewSet()
	for _, host := range curHosts {
		curHostsSet.Add(host)
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

	removeHosts := curHostsSet.Copy()
	removeHosts.Subtract(newHostsSet)

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
	}

	addHosts := newHostsSet.Copy()
	addHosts.Subtract(curHostsSet)

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
	}

	curHosts = hosts

	return
}
package state

import (
	"github.com/pritunl/pritunl-link/config"
)

var (
	DefaultInterface = ""
	DefaultGateway   = ""
	LocalAddress     = ""
	PublicAddress    = ""
	Address6         = ""
	Status           = map[string]map[string]string{}
	IsDirectClient   = false
	DirectIpsecState *State
)

type State struct {
	Id     string  `json:"id"`
	Ipv6   bool    `json:"ipv6"`
	Action string  `json:"action"`
	Type   string  `json:"type"`
	Secret string  `json:"-"`
	Hash   string  `json:"hash"`
	Links  []*Link `json:"links"`
}

type Link struct {
	Hash         string   `json:"hash"`
	PreSharedKey string   `json:"pre_shared_key"`
	Right        string   `json:"right"`
	LeftSubnets  []string `json:"left_subnets"`
	RightSubnets []string `json:"right_subnets"`
}

func GetDefaultInterface() string {
	iface := config.Config.DefaultInterface
	if iface != "" {
		return iface
	}
	return DefaultInterface
}

func GetDefaultGateway() string {
	gateway := config.Config.DefaultGateway
	if gateway != "" {
		return gateway
	}
	return DefaultGateway
}

func GetLocalAddress() string {
	addr := config.Config.LocalAddress
	if addr != "" {
		return addr
	}
	return LocalAddress
}

func GetPublicAddress() string {
	addr := config.Config.PublicAddress
	if addr != "" {
		return addr
	}
	return PublicAddress
}

func GetAddress6() string {
	addr := config.Config.Address6
	if addr != "" {
		return addr
	}
	return Address6
}

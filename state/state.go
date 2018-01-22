package state

import (
	"github.com/pritunl/pritunl-link/config"
)

var (
	DefaultInterface = ""
	LocalAddress     = ""
	PublicAddress    = ""
	Address6         = ""
	Status           = map[string]map[string]string{}
	IsDirectClient   = false
)

type State struct {
	Id     string  `json:"id"`
	Type   string  `json:"type"`
	Secret string  `json:"-"`
	Hash   string  `json:"hash"`
	Links  []*Link `json:"links"`
}

type Link struct {
	PreSharedKey string   `json:"pre_shared_key"`
	Right        string   `json:"right"`
	LeftSubnets  []string `json:"left_subnets"`
	RightSubnets []string `json:"right_subnets"`
}

func GetDefaultInterface() string {
	addr := config.Config.DefaultInterface
	if addr != "" {
		return addr
	}
	return DefaultInterface
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

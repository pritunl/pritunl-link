package state

import (
	"github.com/pritunl/pritunl-link/config"
)

var (
	LocalAddress  = ""
	PublicAddress = ""
)

type State struct {
	Id     string  `json:"id"`
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

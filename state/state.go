package state

import (
	"github.com/pritunl/pritunl-link/config"
)

var PublicAddress = ""

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

func GetPublicAddress() string {
	addr := config.Config.PublicAddress
	if addr != "" {
		return addr
	}
	return PublicAddress
}

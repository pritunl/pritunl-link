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
	WgPrivateKey     = ""
	WgPublicKey      = ""
	Status           = map[string]string{}
	IsDirectClient   = false
	DirectIpsecState *State
)

type State struct {
	Id             string            `json:"id"`
	Mode           string            `json:"mode"`
	Protocol       string            `json:"protocol"`
	WgPort         int               `json:"wg_port"`
	Ipv6           bool              `json:"ipv6"`
	Action         string            `json:"action"`
	Type           string            `json:"type"`
	Cached         bool              `json:"-"`
	Secret         string            `json:"-"`
	Hash           string            `json:"hash"`
	Links          []*Link           `json:"links"`
	Hosts          map[string]string `json:"hosts"`
	PreferredIke   string            `json:"preferred_ike"`
	PreferredEsp   string            `json:"preferred_esp"`
	ForcePreferred bool              `json:"force_preferred"`
}

func (s *State) Copy() *State {
	return &State{
		Id:             s.Id,
		Mode:           s.Mode,
		Protocol:       s.Protocol,
		WgPort:         s.WgPort,
		Ipv6:           s.Ipv6,
		Action:         s.Action,
		Type:           s.Type,
		Cached:         s.Cached,
		Secret:         s.Secret,
		Hash:           s.Hash,
		Links:          s.Links,
		Hosts:          s.Hosts,
		PreferredIke:   s.PreferredIke,
		PreferredEsp:   s.PreferredEsp,
		ForcePreferred: s.ForcePreferred,
	}
}

type Link struct {
	Id           string   `json:"id"`
	Static       bool     `json:"static"`
	Hash         string   `json:"hash"`
	PreSharedKey string   `json:"pre_shared_key"`
	Right        string   `json:"right"`
	WgPublicKey  string   `json:"wg_public_key"`
	LeftSubnets  []string `json:"left_subnets"`
	RightSubnets []string `json:"right_subnets"`
}

func GetStatus(connId string) string {
	status := Status
	connStatus := status[connId]
	if connStatus != "" {
		return connStatus
	}
	return "disconnected"
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

func Init() (err error) {
	privateKey, err := GeneratePrivateKey()
	if err != nil {
		return
	}
	publicKey := privateKey.PublicKey()

	WgPrivateKey = privateKey.String()
	WgPublicKey = publicKey.String()

	return
}

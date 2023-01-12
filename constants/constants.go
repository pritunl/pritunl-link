package constants

import (
	"path"
	"time"
)

const (
	Version                   = "1.0.2606.36"
	VarDir                    = "/var/lib/pritunl_link"
	LogPath                   = "/var/log/pritunl_link.log"
	ConfPath                  = "/etc/pritunl_link.json"
	IpsecConfPath             = "/etc/ipsec.conf"
	IpsecSecretsPath          = "/etc/ipsec.secrets"
	IpsecDirPath              = "/etc/ipsec.pritunl"
	PublicIpServer            = "https://app4.pritunl.com/ip"
	PublicIp6Server           = "https://app6.pritunl.com/ip"
	DefaultDiconnectedTimeout = 30 * time.Second
	DiconnectedTimeoutBackoff = 60 * time.Second
	UpdateAdvertiseRate       = 90
	UpdateAdvertiseReplay     = 15
)

var (
	Interrupt     = false
	RoutesPath    = path.Join(VarDir, "routes")
	CurRoutesPath = path.Join(VarDir, "cur_routes")
)

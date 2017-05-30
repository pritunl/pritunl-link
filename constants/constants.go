package constants

import (
	"path"
	"time"
)

const (
	Version                   = "1.0.553.15"
	VarDir                    = "/var/lib/pritunl_link"
	LogPath                   = "/var/log/pritunl_link.log"
	ConfPath                  = "/etc/pritunl_link.json"
	IpsecConfPath             = "/etc/ipsec.conf"
	IpsecSecretsPath          = "/etc/ipsec.secrets"
	IpsecDirPath              = "/etc/ipsec.pritunl"
	PublicIpServer            = "https://app.pritunl.com/ip"
	PublicIp6Server           = "https://app6.pritunl.com/ip"
	DefaultDiconnectedTimeout = 45 * time.Second
	UpdateAdvertiseRate       = 90
	UpdateAdvertiseReplay     = 15
	StateCacheTtl             = 20 * time.Second
)

var (
	Interrupt     = false
	RoutesPath    = path.Join(VarDir, "routes")
	CurRoutesPath = path.Join(VarDir, "cur_routes")
)

package constants

import (
	"path"
	"time"
)

const (
	Version                   = "1.0.549.42"
	VarDir                    = "/var/lib/pritunl"
	LogPath                   = "/var/log/pritunl.log"
	ConfPath                  = "/etc/pritunl-link.json"
	IpsecConfPath             = "/etc/ipsec.conf"
	IpsecSecretsPath          = "/etc/ipsec.secrets"
	IpsecDirPath              = "/etc/ipsec.pritunl"
	PublicIpServer            = "https://app.pritunl.com/ip"
	PublicIp6Server           = "https://app6.pritunl.com/ip"
	DefaultDiconnectedTimeout = 45 * time.Second
)

var (
	RoutesPath    = path.Join(VarDir, "routes")
	CurRoutesPath = path.Join(VarDir, "cur_routes")
)

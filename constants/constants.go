package constants

import (
	"path"
	"time"
)

const (
	Version            = "1.0.0"
	VarDir             = "/var/lib/pritunl"
	ConfPath           = "/etc/pritunl-link.json"
	PublicIpServer     = "https://app.pritunl.com/ip"
	PublicIp6Server    = "https://app6.pritunl.com/ip"
	DiconnectedTimeout = 60 * time.Second
)

var (
	RoutesPath    = path.Join(VarDir, "routes")
	CurRoutesPath = path.Join(VarDir, "cur_routes")
)

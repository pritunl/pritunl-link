package constants

import (
	"path"
	"time"
)

const (
	Version                   = "1.0.547.31"
	VarDir                    = "/var/lib/pritunl"
	ConfPath                  = "/etc/pritunl-link.json"
	PublicIpServer            = "https://app.pritunl.com/ip"
	PublicIp6Server           = "https://app6.pritunl.com/ip"
	DefaultDiconnectedTimeout = 45 * time.Second
)

var (
	RoutesPath    = path.Join(VarDir, "routes")
	CurRoutesPath = path.Join(VarDir, "cur_routes")
)

// TODO Port option
// TODO Show status in web console
// TODO Count connections show 2/3 for 2 of 3 connected
// TODO Test second antergos failover

package constants

import "path"

const (
	Version  = "1.0.0"
	VarDir   = "/var/lib/pritunl"
	ConfPath = "/etc/pritunl-link.json"
)

var (
	RoutesPath    = path.Join(VarDir, "routes")
	CurRoutesPath = path.Join(VarDir, "cur_routes")
)

package constants

import "path"

const (
	Version = "1.0.0"
	VarDir  = "/var/lib/pritunl"
)

var (
	RoutesPath = path.Join(VarDir, "routes")
)

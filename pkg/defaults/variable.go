package defaults

import (
	"path/filepath"
)

var (
	AppVersion = "dev"

	// Build represents the time at which the binary was built
	BuildTime  = ""
	AppCommit  = "0000000"
	ConfigPath = filepath.Join("c:/", "etc", "rancher", "wins", "config")
	CertPath   = filepath.Join("c:/", "etc", "rancher", "agent", "ranchercert")
)

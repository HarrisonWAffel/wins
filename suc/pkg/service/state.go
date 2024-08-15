package service

import "github.com/rancher/wins/cmd/server/config"

// InitialState represents the configuration of
// rancher-wins and rke2 before any changes are made.
// In the event of an error during reconfiguration of the
// service or the related binaries, this struct should be used
// to roll back all changes.
type InitialState struct {
	InitialConfig *config.Config

	// TODO
	//InitialWinsStartType     StartType
	//InitialRke2StartType     StartType
	//InitialServiceDependency bool
}

func BuildInitialState() (InitialState, error) {
	cfg, err := loadConfig("")
	if err != nil {
		return InitialState{}, err
	}

	return InitialState{
		InitialConfig: cfg,
	}, nil
}

func RestoreInitialState(state InitialState) error {
	return saveConfig(state.InitialConfig, "")
}

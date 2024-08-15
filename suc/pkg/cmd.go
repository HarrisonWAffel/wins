package pkg

import (
	"errors"
	"os"

	"github.com/rancher/wins/suc/pkg/service"
	"github.com/urfave/cli/v2"
)

func Run(ctx *cli.Context) error {
	var errs []error
	refreshService := false

	// commands should be run in a specific order to avoid restarting the service unnecessarily.
	// 1. Config file updates
	// 2. Service start type updates
	// 3. Service dependencies
	// 4. Any binary updates required
	//   4a. The service(s) must be stopped before this can occur
	// 5. start / restart the relevant services

	if ctx.Bool("update-config") || os.Getenv("CATTLE_WINS_UPDATE_CONFIG") != "" {
		// update the config using the env vars
		restartServiceDueToConfigChange, err := service.UpdateConfigFromEnvVars("")
		refreshService = restartServiceDueToConfigChange
		if err != nil {
			errs = append(errs, err)
		}
	}

	if refreshService && (errs == nil || len(errs) == 0) {
		if err := service.RefreshWinsService(); err != nil {
			return err
		}
	}

	return combineErrors(errs)
}

func combineErrors(errs []error) error {
	var err error
	for _, e := range errs {
		err = errors.Join(err, e)
	}
	return err
}

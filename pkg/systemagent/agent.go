package systemagent

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rancher/system-agent/pkg/applyinator"
	"github.com/rancher/system-agent/pkg/config"
	"github.com/rancher/system-agent/pkg/image"
	"github.com/rancher/system-agent/pkg/k8splan"
	"github.com/rancher/system-agent/pkg/localplan"
	"github.com/rancher/system-agent/pkg/version"
	"github.com/rancher/wins/pkg/network"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

type Agent struct {
	cfg *config.AgentConfig
}

func (a *Agent) Run(ctx context.Context) error {

	if a.cfg == nil {
		logrus.Info("Rancher System Agent configuration not found, not starting system agent.")
		return nil
	}

	logrus.Infof("Rancher System Agent version %s is starting", version.FriendlyVersion())

	if !a.cfg.LocalEnabled && !a.cfg.RemoteEnabled {
		return errors.New("local and remote were both not enabled. exiting, as one must be enabled")
	}

	logrus.Infof("Setting %s as the working directory", a.cfg.WorkDir)

	imageUtil := image.NewUtility(a.cfg.ImagesDir, a.cfg.ImageCredentialProviderConfig, a.cfg.ImageCredentialProviderBinDir, a.cfg.AgentRegistriesFile)
	applier := applyinator.NewApplyinator(a.cfg.WorkDir, a.cfg.PreserveWorkDir, a.cfg.AppliedPlanDir, imageUtil)

	if a.cfg.RemoteEnabled {
		logrus.Infof("Starting remote watch of plans")

		var connInfo config.ConnectionInfo

		if err := config.Parse(a.cfg.ConnectionInfoFile, &connInfo); err != nil {
			return fmt.Errorf("unable to parse connection info file: %v", err)
		}

		kc, err := clientcmd.RESTConfigFromKubeConfig([]byte(connInfo.KubeConfig))
		if err != nil {
			return fmt.Errorf("unable to parse kubeconfig from connection info file: %w", err)
		}

		// We need to ensure we can reach the host before we start any wrangler controllers.
		// Windows services start asynchronously, and wins may have started before other core services which
		// could result in an inability to reach the rancher host. This check allows us to wait for a few seconds
		// so we don't prematurely declare an inability to connect
		retryLimit := 5
		if !network.EnsureHostIsReachable(kc, retryLimit) {
			return fmt.Errorf("could not establish a connection with the rancher host after %d attempts", retryLimit)
		}

		k8splan.Watch(ctx, *applier, connInfo)
	}

	if a.cfg.LocalEnabled {
		logrus.Infof("Starting local watch of plans in %s", a.cfg.LocalPlanDir)
		localplan.WatchFiles(ctx, *applier, a.cfg.LocalPlanDir)
	}

	return nil
}

func New(cfg *config.AgentConfig) *Agent {
	return &Agent{
		cfg: cfg,
	}
}

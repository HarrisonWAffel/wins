package service

import (
	"fmt"
	"github.com/rancher/wins/pkg/csiproxy"
	"os"
	"strconv"
	"strings"

	"github.com/rancher/wins/cmd/server/config"
	"github.com/sirupsen/logrus"
)

var (

	// TODO: Review the applicability of these variables.
	//       How much should be controllable?

	ConfigDirEnvVar       = envVar{varName: "wins_config_dir"}
	DebugEnvVar           = envVar{varName: "debug"}
	ListenEnvVar          = envVar{varName: "listen"}
	ProxyEnvVar           = envVar{varName: "proxy"}
	AgentStringTLSEnvVar  = envVar{varName: "strict_verify"}
	CsiProxyURLEnvVar     = envVar{varName: "csi_proxy_url"}
	CsiProxyVersionEnvVar = envVar{varName: "csi_proxy_version"}
	KubeletPathEnvVar     = envVar{varName: "kubelet_path"}
)

const (
	defaultConfigFile = "C:/etc/rancher/wins/config"
)

type envVar struct {
	varName string
}

func (e *envVar) get() string {
	return os.Getenv(e.name())
}

func (e *envVar) getSimple() string {
	return os.Getenv(e.simpleName())
}

func (e *envVar) name() string {
	return fmt.Sprintf("CATTLE_WINS_%s", strings.ToUpper(e.varName))
}

func (e *envVar) simpleName() string {
	return strings.ToUpper(e.varName)
}

func loadConfig(path string) (*config.Config, error) {
	configDirEnv := os.Getenv(ConfigDirEnvVar.get())
	if path == "" && configDirEnv != "" {
		path = configDirEnv
	}
	if path == "" {
		path = defaultConfigFile
	}

	cfg := config.DefaultConfig()
	err := config.LoadConfig(path, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func saveConfig(cfg *config.Config, path string) error {
	configDirEnv := os.Getenv(ConfigDirEnvVar.get())
	if path == "" && configDirEnv != "" {
		path = configDirEnv
	}
	if path == "" {
		path = defaultConfigFile
	}

	return config.SaveConfig(path, cfg)
}

func UpdateConfigFromEnvVars(path string) (bool, error) {
	logrus.Infof("Loading config from host")
	cfg, err := loadConfig(path)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %v", err)
	}

	configNeedsUpdate := false
	logrus.Infof("Checking for %s value. This is a boolean flag, expecting 'true'", DebugEnvVar.name())
	if v := DebugEnvVar.get(); v != "" {
		logrus.Infof("Found value '%s' for %s", v, DebugEnvVar.name())
		givenBool := strings.ToLower(DebugEnvVar.get()) == "true"
		if cfg.Debug != givenBool {
			cfg.Debug = givenBool
			configNeedsUpdate = true
		}
	}

	logrus.Infof("Checking for %s value", ListenEnvVar.name())
	if v := ListenEnvVar.get(); v != "" && cfg.Listen != v {
		logrus.Infof("Found value '%s' for %s", v, ListenEnvVar.name())
		cfg.Listen = v
		configNeedsUpdate = true
	}

	logrus.Infof("Checking for %s value", ProxyEnvVar.name())
	if v := ProxyEnvVar.get(); v != "" && cfg.Proxy != v {
		logrus.Infof("Found value '%s' for %s", v, ProxyEnvVar.name())
		cfg.Proxy = v
		configNeedsUpdate = true
	}

	logrus.Infof("Checking for %s value. This is a boolean flag, expecting 'true'", AgentStringTLSEnvVar.simpleName())
	if v := AgentStringTLSEnvVar.getSimple(); v != "" {
		logrus.Infof("Found value '%s' for %s", v, AgentStringTLSEnvVar.simpleName())
		tlsMode, err := strconv.ParseBool(strings.ToLower(v))
		if err != nil {
			logrus.Errorf("Error encountered while pasring %s, field will not be updated: %v", AgentStringTLSEnvVar.getSimple(), err)
		} else if cfg.AgentStrictTLSMode != tlsMode {
			cfg.AgentStrictTLSMode = tlsMode
			configNeedsUpdate = true
		}
	}

	logrus.Infof("Checking for %s value", CsiProxyURLEnvVar.name())
	if v := CsiProxyURLEnvVar.get(); v != "" {
		if cfg.CSIProxy == nil {
			cfg.CSIProxy = &csiproxy.Config{}
		}
		logrus.Infof("Found value '%s' for %s", v, CsiProxyURLEnvVar.name())
		cfg.CSIProxy.URL = v
		configNeedsUpdate = true
	}

	logrus.Infof("Checking for %s value", CsiProxyVersionEnvVar.name())
	if v := CsiProxyVersionEnvVar.get(); v != "" && cfg.CSIProxy.Version != v {
		logrus.Infof("Found value '%s' for %s", v, CsiProxyVersionEnvVar.name())
		cfg.CSIProxy.Version = v
		configNeedsUpdate = true
	}

	logrus.Infof("Checking for %s value", KubeletPathEnvVar.name())
	if v := KubeletPathEnvVar.get(); v != "" && cfg.CSIProxy.KubeletPath != v {
		logrus.Infof("Found value '%s' for %s", v, KubeletPathEnvVar.name())
		cfg.CSIProxy.KubeletPath = v
		configNeedsUpdate = true
	}

	// If we haven't made any changes there is no reason to update the config file
	if configNeedsUpdate {
		logrus.Infof("Detected a change in configuration, updating config file")
		err = saveConfig(cfg, path)
		if err != nil {
			return configNeedsUpdate, fmt.Errorf("failed to save config: %v", err)
		}
	} else {
		logrus.Infof("Did not detect a change in configuration")
	}

	return configNeedsUpdate, nil
}

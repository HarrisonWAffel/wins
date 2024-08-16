//go:build windows

package service

import (
	"encoding/json"
	"github.com/rancher/wins/pkg/csiproxy"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/rancher/wins/cmd/server/config"
	v1 "k8s.io/api/core/v1"
)

const (
	// configFileLoc denotes where the test
	// config will be placed on disk when running in GHA.
	// The drive letter is intentionally omitted.
	configFileLoc = "./wins-test-config"
)

func setupTest(vars []v1.EnvVar, t *testing.T) {
	for _, evar := range os.Environ() {
		if !strings.Contains(evar, "CATTLE") {
			continue
		}
		err := os.Unsetenv(strings.Split(evar, "=")[0])
		if err != nil {
			t.Fatalf("failed to clear environment variable %s: %v", evar, err)
		}
	}
	for _, evar := range vars {
		err := os.Setenv(evar.Name, evar.Value)
		if err != nil {
			t.Fatalf("failed to set environment variable %s: %v", evar.Name, err)
		}
	}
	err := os.Setenv(ConfigDirEnvVar.name(), "./config")
	if err != nil {
		t.Fatalf("Could not set %s", ConfigDirEnvVar.name())
	}
	err = os.Remove(configFileLoc)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unable to remove existing config file")
	}
}

func Test_UpdateConfigFromEnvVars(t *testing.T) {
	type test struct {
		name           string
		envVars        []v1.EnvVar
		expectedConfig func() *config.Config
		updateExpected bool
	}

	tests := []test{
		{
			name: "Update single known field",
			envVars: []v1.EnvVar{
				{
					Name:  DebugEnvVar.name(),
					Value: "true",
				},
			},
			expectedConfig: func() *config.Config {
				def := config.DefaultConfig()
				def.Debug = true
				return def
			},
			updateExpected: true,
		},
		{
			name:    "No update required",
			envVars: []v1.EnvVar{},
			expectedConfig: func() *config.Config {
				return config.DefaultConfig()
			},
			updateExpected: false,
		},
		{
			name: "No update due to unknown env var",
			envVars: []v1.EnvVar{
				{
					Name:  "Unknown",
					Value: "variable",
				},
			},
			expectedConfig: func() *config.Config {
				return config.DefaultConfig()
			},
			updateExpected: false,
		},
		{
			name: "Update many known fields",
			envVars: []v1.EnvVar{
				{
					Name:  DebugEnvVar.name(),
					Value: "true",
				},
				{
					Name:  AgentStringTLSEnvVar.simpleName(),
					Value: "true",
				},
				{
					Name:  CsiProxyVersionEnvVar.name(),
					Value: "v1.2.3",
				},
				{
					Name:  CsiProxyURLEnvVar.name(),
					Value: "https://fake.com",
				},
			},
			expectedConfig: func() *config.Config {
				def := config.DefaultConfig()
				def.Debug = true
				def.AgentStrictTLSMode = true
				def.CSIProxy = &csiproxy.Config{
					Version: "v1.2.3",
					URL:     "https://fake.com",
				}
				return def
			},
			updateExpected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupTest(tc.envVars, t)
			expectedConfig := tc.expectedConfig()

			updated, err := UpdateConfigFromEnvVars(configFileLoc)
			if err != nil {
				t.Logf("UpdateConfigFromEnvVars returned an unexpected error: %v", err)
				t.FailNow()
			}

			if updated && !tc.updateExpected {
				j, _ := json.MarshalIndent(os.Environ(), "", " ")
				t.Logf("Config was updated unexpectedly when the following env vars were used: %s", string(j))
				t.FailNow()
			}

			updatedConfig := config.DefaultConfig()
			updateConfigErr := config.LoadConfig(configFileLoc, updatedConfig)
			if updateConfigErr != nil {
				t.Logf("encountered an error when reloading the config file: %v", err)
				t.FailNow()
			}

			if !reflect.DeepEqual(expectedConfig, updatedConfig) {
				j1, _ := json.MarshalIndent(expectedConfig, "", " ")
				j2, _ := json.MarshalIndent(updatedConfig, "", " ")
				t.Logf("Expected config did not match updated config.\nExpected: %s\nUpdated: %s", string(j1), string(j2))
				t.Fail()
			}
		})
	}
}

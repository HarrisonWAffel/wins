package host

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rancher/wins/pkg/defaults"
	"github.com/sirupsen/logrus"
)

const (
	defaultWinsPath         = "c:\\Windows\\wins.exe"
	winsUsrLocalBinPath     = "c:\\usr\\local\\bin\\wins.exe"
	skipBinaryUpgradeEnvVar = "CATTLE_WINS_SKIP_BINARY_UPGRADE"
	defaultConfigDir        = "c:\\etc\\rancher\\wins"
)

// UpgradeRancherWinsBinary will attempt to upgrade the wins.exe binary installed on the host.
// The version to be installed is embedded within the SUC binary, located in the winsBinary variable.
// Upgrades will only be attempted if the CATTLE_WINS_SKIP_BINARY_UPGRADE environment variable is not set to 'true',
// and the currently installed version differs from the one embedded (determined by the output of 'wins.exe --version').
// During an upgrade attempt the rancher-wins service will be temporarily stopped.
// A boolean is returned to indicate if the rancher-wins service needs to be restarted due to a successful upgrade.
func UpgradeRancherWinsBinary() (bool, error) {
	if strings.ToLower(os.Getenv(skipBinaryUpgradeEnvVar)) == "true" ||
		strings.ToLower(os.Getenv(skipBinaryUpgradeEnvVar)) == "$true" {
		logrus.Warnf("environment variable '%s' was set to true, will not attempt to upgrade binary", skipBinaryUpgradeEnvVar)
		return false, nil
	}

	currentVersion, err := GetRancherWinsVersionFromBinary(defaultWinsPath)
	if err != nil {
		return false, fmt.Errorf("could not determine current wins.exe version: %w", err)
	}

	// we use the AppVersion built into the binary during compilation to indicate
	// the version of wins.exe that should be installed. See magetools/gotool.go for more information.
	desiredVersion := defaults.AppVersion

	// We should never install a dirty version of wins.exe onto a host.
	if strings.Contains(desiredVersion, "-dirty") {
		return false, fmt.Errorf("cannot upgrade wins.exe version, refuse to install embedded dirty version (version: %s)", desiredVersion)
	}

	restartService, upgradeErr := WinsLocalUpgrader{
		desiredVersion: desiredVersion,
		currentVersion: currentVersion,
	}.UpgradeWins()

	if upgradeErr != nil {
		return false, upgradeErr
	}

	return restartService, nil
}

// GetRancherWinsVersionFromBinary executes the wins.exe binary located at 'path' and passes the '--version'
// flag. The release version or commit hash is returned. If the binary returns expected output,
// was built with a dirty commit, or does not exist, an error will be returned.
func GetRancherWinsVersionFromBinary(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("must specify a path")
	}

	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("provided path (%s) does not exist", path)
		}
		return "", fmt.Errorf("encoutered error stat'ing '%s': %w", path, err)
	}

	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		logrus.Errorf("could not invoke %s to determine installed rancher-wins version: %v", path, err)
		return "", fmt.Errorf("failed to invoke %s: %w", path, err)
	}

	logrus.Debugf("'%s --version' output: %s", path, string(out))
	return parseWinsVersion(string(out))
}

func parseWinsVersion(winsOutput string) (string, error) {

	// Expected output format is 'rancher-wins version v0.x.y[-rc.z]'"
	// A dirty binary will return 'rancher-wins version COMMIT-dirty'
	// A non-tagged version will return 'rancher-wins version COMMIT'
	s := strings.Split(winsOutput, " ")
	if len(s) != 3 {
		return "", fmt.Errorf("'wins.exe --version' did not return expected output ('%s' was returned)", s)
	}

	verString := strings.Trim(s[2], "\n")

	logrus.Debugf("Detected wins.exe version '%s'", verString)

	// We should error out if the binary we're working with is dirty, but
	// if it's simply untagged we should proceed with the upgrade.
	if strings.Contains(verString, "dirty") {
		return "", fmt.Errorf("wins.exe binary returned a dirty version (%s)", verString)
	}

	logrus.Debugf("wins.exe version %s is a release version: %t", verString, strings.Contains(verString, "."))
	return verString, nil
}

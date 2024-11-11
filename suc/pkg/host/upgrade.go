package host

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rancher/wins/suc/pkg/service"
	"github.com/sirupsen/logrus"
)

type WinsLocalUpgrader struct {
	desiredVersion string
	currentVersion string
}

func (wlu WinsLocalUpgrader) UpgradeWins() (bool, error) {
	// determine if the installed version is already up-to-date
	curVer, err := GetRancherWinsVersionFromBinary(defaultWinsPath)
	if err != nil {
		return false, fmt.Errorf("failed to get version of updated wins.exe binary")
	}

	if curVer == wlu.desiredVersion {
		logrus.Infof("Installed wins.exe version is already up-to-date (%s)", curVer)
		return false, nil
	}

	logrus.Infof("Attempting to upgrade wins.exe to version %s", wlu.desiredVersion)

	err = wlu.stageEmbeddedBinary()
	if err != nil {
		return false, fmt.Errorf("failed to stage embedded wins.exe binary onto the host: %w", err)
	}

	return true, nil
}

// stageEmbeddedBinary writes the embedded binary onto the disk in the /etc/rancher/wins directory.
// Once written, the binary is invoked to ensure that it is not corrupted and is running the expected version.
// After confirming the version, the old wins.exe binary to moved to the /etc/rancher/wins directory, and the updated
// binary is moved into the /usr/local/bin and c:\Windows directories.
// Once the upgraded binary has been moved into place, it is invoked once again to confirm the file was copied correctly.
// Finally, the old wins.exe binary is deleted from the disk.
func (wlu WinsLocalUpgrader) stageEmbeddedBinary() error {
	logrus.Info("Writing updated wins.exe to disk")
	// write the embedded binary to disk
	updatedBinaryPath := getPathForVersion(defaultConfigDir, wlu.desiredVersion)
	err := os.WriteFile(updatedBinaryPath, winsBinary, os.ModePerm)
	if err != nil {
		return err
	}

	// confirm that the new binary works and returns the version that we expect
	updatedVer, err := GetRancherWinsVersionFromBinary(updatedBinaryPath)
	if err != nil {
		return fmt.Errorf("was not able to determine version of %s: %w", updatedBinaryPath, err)
	}

	if updatedVer != wlu.desiredVersion {
		err = os.Remove(updatedBinaryPath)
		if err != nil {
			return fmt.Errorf("%s did not return expected version (desired version: %s, returned version: %s): failed to delete updated binary %s: %w", updatedBinaryPath, wlu.desiredVersion, updatedVer, updatedVer, err)
		}
		return fmt.Errorf("%s did not return expected version (desired version: %s, returned version: %s)", updatedBinaryPath, wlu.desiredVersion, updatedVer)
	}

	logrus.Info("Stopping rancher-wins...")
	rw, _, err := service.OpenRancherWinsService()
	if err != nil {
		return fmt.Errorf("failed to open rancher-wins service while attempting to upgrade binary: %w", err)
	}

	// The service needs to be stopped before we can modify
	// the binary it uses
	err = rw.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop rancher-wins service while attempting to upgrade binary: %w", err)
	}

	logrus.Info("Staging wins.exe binaries")
	// move the old binary tp /etc/rancher/wins and rename it to include its version
	oldBinaryPath := getPathForVersion(defaultConfigDir, wlu.currentVersion)
	logrus.Infof("Moving %s to %s", defaultWinsPath, oldBinaryPath)
	if err = os.Rename(defaultWinsPath, oldBinaryPath); err != nil {
		return fmt.Errorf("failed to rename existing win.exe binary to %s: %w", oldBinaryPath, err)
	}

	logrus.Infof("Copying %s to %s", updatedBinaryPath, defaultWinsPath)
	// we store wins in the following directories. Note that the rancher-wins
	// service looks for wins.exe in C:\Windows, but for consistencies sake
	// we should also ensure it's updated in /usr/local/bin
	err = copyFile(updatedBinaryPath, defaultWinsPath)
	if err != nil {
		return fmt.Errorf("failed to copy new wins.exe binary to %s: %w", defaultWinsPath, err)
	}

	logrus.Infof("Moving %s to %s", updatedBinaryPath, winsUsrLocalBinPath)
	err = os.Rename(updatedBinaryPath, winsUsrLocalBinPath)
	if err != nil {
		return fmt.Errorf("failed to rename new wins.exe binary to %s: %w", winsUsrLocalBinPath, err)
	}

	logrus.Infof("Validating updated binary...")
	// confirm that the renamed binary is running the desired version
	newVer, err := GetRancherWinsVersionFromBinary(defaultWinsPath)
	if err != nil {
		return fmt.Errorf("failed to get version of updated wins.exe binary")
	}

	if newVer != wlu.desiredVersion {
		return fmt.Errorf("failed to verify version of updated wins.exe binary (returned version: %s, desired version: %s)", newVer, wlu.desiredVersion)
	}

	logrus.Infof("Removing out-dated wins.exe binary (%s)", oldBinaryPath)
	// clean up old binary
	err = os.Remove(oldBinaryPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error encountered deleting old wins.exe binary: %w", err)
	}

	logrus.Infof("Successfully upgraded wins.exe to version %s", wlu.desiredVersion)
	return nil
}

// copyFile opens the file located at 'source' and creates a new file at 'destination'
// with the same contents. Note that permission bits on Windows do not function in the same
// way as Linux, the owner bit is copied to all other bits. The caller of copyFile must
// ensure that the destination is covered by appropriate access control lists.
func copyFile(source, dest string) error {
	_, err := os.Stat(source)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	err = os.WriteFile(dest, b, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func getPathForVersion(basePath, version string) string {
	return fmt.Sprintf("%s/wins-%s.exe", basePath, strings.Trim(version, "\n"))
}
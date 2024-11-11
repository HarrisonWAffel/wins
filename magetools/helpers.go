package magetools

import (
	"os"
	"strings"

	"github.com/magefile/mage/sh"
)

func IsGitClean() (bool, error) {
	result, err := sh.Output("git", "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	if result != "" {
		return false, nil
	}
	return true, nil
}

func GetLatestTag() (string, error) {
	result, err := sh.Output("git", "tag", "-l", "--contains", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func GetCommit() (string, error) {
	result, err := sh.Output("git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func CopyFile(source, dest string) error {
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

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

func home() string { h, _ := os.UserHomeDir(); return h }

func defaultUserDataDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "google-chrome")
	}
	return filepath.Join(home(), ".config", "google-chrome")
}

func installDir() string {
	return filepath.Join(home(), ".local", "share", "chrome-profile-manager")
}

func quitChromeNicely(udd string) bool { return false }

func launchChrome(profileDir string) error {
	args := []string{"--profile-directory=" + profileDir, "--restore-last-session"}
	if cfg, _ := loadConfigLoose(); cfg.UserDataDir != defaultUserDataDir() {
		args = append(args, "--user-data-dir="+cfg.UserDataDir)
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return spawnDetached(p, args...)
		}
	}
	return errors.New("could not find Chrome to reopen it")
}

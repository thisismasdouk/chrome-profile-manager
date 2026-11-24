package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func home() string { h, _ := os.UserHomeDir(); return h }

func defaultUserDataDir() string {
	return filepath.Join(home(), "Library", "Application Support", "Google", "Chrome")
}

func installDir() string {
	return filepath.Join(home(), "Library", "Application Support", "ChromeProfileManager")
}

func quitChromeNicely(udd string) bool {
	if udd != defaultUserDataDir() {
		return false
	}
	return exec.Command("osascript", "-e", `tell application id "com.google.Chrome" to quit`).Run() == nil
}

func launchChrome(profileDir string) error {
	args := []string{"-n", "-a", "Google Chrome", "--args", "--profile-directory=" + profileDir, "--restore-last-session"}
	if cfg, _ := loadConfigLoose(); cfg.UserDataDir != defaultUserDataDir() {
		args = append(args, "--user-data-dir="+cfg.UserDataDir)
	}
	return exec.Command("open", args...).Run()
}

//go:build darwin || linux

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// chromePID reads Chrome's SingletonLock ("hostname-pid") in the data folder.
func chromePID(udd string) int {
	target, err := os.Readlink(filepath.Join(udd, "SingletonLock"))
	if err != nil {
		return 0
	}
	i := strings.LastIndex(target, "-")
	if i < 0 {
		return 0
	}
	pid, _ := strconv.Atoi(target[i+1:])
	return pid
}

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func chromeRunning(udd string) bool { return pidAlive(chromePID(udd)) }

func waitForExit(udd string, d time.Duration) bool {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if !chromeRunning(udd) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return !chromeRunning(udd)
}

func quitChrome(udd string) {
	if quitChromeNicely(udd) && waitForExit(udd, 30*time.Second) {
		return
	}
	if pid := chromePID(udd); pidAlive(pid) {
		syscall.Kill(pid, syscall.SIGTERM)
		waitForExit(udd, 30*time.Second)
	}
}

func spawnDetached(exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func registerHost(manifestFile string, manifest []byte, udd string) error {
	dir := filepath.Join(udd, "NativeMessagingHosts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, hostName+".json"), manifest, 0o644)
}

func unregisterHost(udd string) {
	os.Remove(filepath.Join(udd, "NativeMessagingHosts", hostName+".json"))
}

func registerUninstaller(exe string) {}
func unregisterUninstaller()         {}

func removeSelfLater(dir string) { os.RemoveAll(dir) }

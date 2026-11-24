package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func helperFileName() string {
	if runtime.GOOS == "windows" {
		return "profile-helper.exe"
	}
	return "profile-helper"
}

func runInstall(udd string, interactive bool) {
	err := install(udd)
	if err != nil {
		fmt.Println()
		fmt.Println("  Something went wrong:", err)
		fmt.Println()
	} else {
		fmt.Println()
		fmt.Println("  All set! The Profile Manager helper is installed.")
		fmt.Println("  Next: load the extension in Chrome (see the README), then click its icon.")
		fmt.Println()
	}
	if interactive || runtime.GOOS == "windows" {
		fmt.Print("  Press Enter to close this window...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
	if err != nil {
		os.Exit(1)
	}
}

func install(udd string) error {
	if udd == "" {
		udd = defaultUserDataDir()
	}
	if _, err := os.Stat(filepath.Join(udd, "Local State")); err != nil {
		return fmt.Errorf("could not find Google Chrome's data folder at %s. Is Chrome installed and has it been opened at least once?", udd)
	}
	dir := installDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	src, err := os.Executable()
	if err != nil {
		return err
	}
	src, _ = filepath.EvalSymlinks(src)
	dst := filepath.Join(dir, helperFileName())
	if !sameFile(src, dst) {
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("could not copy the helper into %s (if Chrome is open, close it completely and try again): %v", dir, err)
		}
	}
	os.Chmod(dst, 0o755)
	cfg, _ := json.MarshalIndent(Config{UserDataDir: udd}, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "config.json"), cfg, 0o644); err != nil {
		return err
	}
	manifest := map[string]any{
		"name":            hostName,
		"description":     "Profile Manager helper (lists and deletes Chrome profiles)",
		"path":            dst,
		"type":            "stdio",
		"allowed_origins": allowedOrigins(),
	}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	manifestFile := filepath.Join(dir, hostName+".json")
	if err := os.WriteFile(manifestFile, mb, 0o644); err != nil {
		return err
	}
	fmt.Println("  Helper installed to:", dir)
	fmt.Println("  Chrome data folder:", udd)
	if err := registerHost(manifestFile, mb, udd); err != nil {
		return err
	}
	registerUninstaller(dst)
	return nil
}

func runUninstall() {
	cfg, _ := loadConfigLoose()
	unregisterHost(cfg.UserDataDir)
	unregisterUninstaller()
	dir := installDir()
	for _, f := range []string{"config.json", hostName + ".json", "pending-deletion.json", "worker.alive", "last-result.json", "helper.log"} {
		os.Remove(filepath.Join(dir, f))
	}
	if exe, _ := os.Executable(); !sameFile(exe, filepath.Join(dir, helperFileName())) {
		os.Remove(filepath.Join(dir, helperFileName()))
		os.Remove(dir)
	} else {
		removeSelfLater(dir)
	}
	fmt.Println("  Profile Manager helper removed. You can now remove the extension from chrome://extensions.")
	if runtime.GOOS == "windows" {
		fmt.Print("  Press Enter to close this window...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func loadConfigLoose() (*Config, error) {
	cfg := &Config{}
	if b, err := os.ReadFile(filepath.Join(installDir(), "config.json")); err == nil {
		json.Unmarshal(b, cfg)
	}
	if cfg.UserDataDir == "" {
		cfg.UserDataDir = defaultUserDataDir()
	}
	return cfg, nil
}

func sameFile(a, b string) bool {
	sa, err1 := os.Stat(a)
	sb, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(sa, sb)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	out.Close()
	return os.Rename(tmp, dst)
}

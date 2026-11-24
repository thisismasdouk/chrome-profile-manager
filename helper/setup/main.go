//go:build windows

// Windows setup program for the Profile Manager helper.
//
// It contains the helper (profile-helper.exe), installs it for the current
// user (no admin rights needed), registers it with Chrome, and shows a
// message box with the result. Build steps are in BUILD.md.
package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload/profile-helper.exe
var helperExe []byte

const title = "Profile Manager helper"

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	messageBoxW = user32.NewProc("MessageBoxW")
)

const (
	mbOK           = 0x0
	mbOKCancel     = 0x1
	mbIconInfo     = 0x40
	mbIconError    = 0x10
	mbIconQuestion = 0x20
	idOK           = 1
)

func msgBox(text string, flags uintptr) int {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(title)
	r, _, _ := messageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	return int(r)
}

func main() {
	if msgBox("This installs the small helper program that lets the Profile Manager extension list and delete your Chrome profiles.\n\nIt installs only for your Windows account and doesn't need administrator rights.\n\nInstall now?", mbOKCancel|mbIconQuestion) != idOK {
		return
	}
	tmp, err := os.MkdirTemp("", "profile-manager-setup-")
	if err != nil {
		fail(err.Error())
		return
	}
	defer os.RemoveAll(tmp)
	exe := filepath.Join(tmp, "profile-helper.exe")
	if err := os.WriteFile(exe, helperExe, 0o755); err != nil {
		fail(err.Error())
		return
	}
	cmd := exec.Command(exe, "--install-quiet")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		fail(msg)
		return
	}
	msgBox("All set! The helper is installed.\n\nGo back to Chrome. The Profile Manager page will notice it within a few seconds (or click Refresh).\n\nTo remove it later: Windows Settings > Apps > \"Profile Manager helper (for Chrome)\".", mbOK|mbIconInfo)
}

func fail(msg string) {
	msgBox("Sorry, the helper could not be installed:\n\n"+msg, mbOK|mbIconError)
}

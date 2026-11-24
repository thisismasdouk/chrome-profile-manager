package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	createNoWindow         = 0x08000000
	detachedProcess        = 0x00000008
	createNewProcessGroup  = 0x00000200
	createBreakawayFromJob = 0x01000000
)

func localAppData() string {
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return v
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "AppData", "Local")
}

func defaultUserDataDir() string {
	return filepath.Join(localAppData(), "Google", "Chrome", "User Data")
}

func installDir() string { return filepath.Join(localAppData(), "ChromeProfileManager") }

// hidden runs a console tool without flashing a window.
func hidden(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return c
}

func chromeRunning(udd string) bool {
	out, err := hidden("tasklist", "/FI", "IMAGENAME eq chrome.exe", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return true // be careful: if we cannot tell, assume it is still running
	}
	return strings.Contains(strings.ToLower(string(out)), "\"chrome.exe\"")
}

func waitForExit(udd string, d time.Duration) bool {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if !chromeRunning(udd) {
			return true
		}
		time.Sleep(time.Second)
	}
	return !chromeRunning(udd)
}

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	procGetClassNameW       = user32.NewProc("GetClassNameW")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
)

func chromePIDs() map[uint32]bool {
	pids := map[uint32]bool{}
	out, err := hidden("tasklist", "/FI", "IMAGENAME eq chrome.exe", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return pids
	}
	for _, ln := range strings.Split(string(out), "\n") {
		f := strings.Split(ln, "\",\"")
		if len(f) > 1 && strings.EqualFold(strings.Trim(f[0], "\" \r"), "chrome.exe") {
			var pid uint32
			fmt.Sscanf(f[1], "%d", &pid)
			if pid > 0 {
				pids[pid] = true
			}
		}
	}
	return pids
}

// endSession tells Chrome that Windows is shutting down. Chrome then saves
// every open window and tab (like a normal Exit from the menu) and quits,
// so the tabs can be restored when it is reopened.
func endSession() bool {
	pids := chromePIDs()
	if len(pids) == 0 {
		return false
	}
	var target uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if !pids[pid] {
			return 1
		}
		if v, _, _ := procIsWindowVisible.Call(hwnd); v == 0 {
			return 1
		}
		buf := make([]uint16, 64)
		procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 64)
		if syscall.UTF16ToString(buf) == "Chrome_WidgetWin_1" {
			target = hwnd
			return 0
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	if target == 0 {
		return false
	}
	const wmQueryEndSession, wmEndSession, smtoAbortIfHung = 0x11, 0x16, 0x2
	var res uintptr
	procSendMessageTimeoutW.Call(target, wmQueryEndSession, 0, 0, smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&res)))
	procSendMessageTimeoutW.Call(target, wmEndSession, 1, 0, smtoAbortIfHung, 10000, uintptr(unsafe.Pointer(&res)))
	return true
}

func quitChrome(udd string) {
	// Best: a "Windows is shutting down" message, which saves all tabs.
	if endSession() && waitForExit(udd, 15*time.Second) {
		logf("Chrome closed (session saved)")
		return
	}
	// Otherwise close politely (like clicking the X on every window).
	for i := 0; i < 6; i++ {
		hidden("taskkill", "/IM", "chrome.exe").Run()
		if waitForExit(udd, 5*time.Second) {
			return
		}
	}
	// Chrome kept running (often "continue running background apps"); force it.
	logf("Chrome did not close politely; forcing it")
	hidden("taskkill", "/F", "/T", "/IM", "chrome.exe").Run()
	waitForExit(udd, 15*time.Second)
}

func spawnDetached(exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createBreakawayFromJob | detachedProcess | createNewProcessGroup,
	}
	err := cmd.Start()
	if err == nil {
		return cmd.Process.Release()
	}
	// Chrome may run us inside a job that forbids breaking away. Ask Windows
	// (WMI) to start the process instead, which places it outside Chrome.
	line := `"` + exe + `"`
	for _, a := range args {
		line += ` "` + a + `"`
	}
	ps := fmt.Sprintf(`$r = Invoke-CimMethod -ClassName Win32_Process -MethodName Create -Arguments @{CommandLine='%s'}; exit $r.ReturnValue`,
		strings.ReplaceAll(line, "'", "''"))
	if err2 := hidden("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run(); err2 != nil {
		return fmt.Errorf("%v; fallback: %v", err, err2)
	}
	return nil
}

func chromeExe() string {
	cands := []string{
		filepath.Join(os.Getenv("ProgramFiles"), `Google\Chrome\Application\chrome.exe`),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Google\Chrome\Application\chrome.exe`),
		filepath.Join(localAppData(), `Google\Chrome\Application\chrome.exe`),
	}
	for _, key := range []string{
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe`,
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe`,
	} {
		if out, err := hidden("reg", "query", key, "/ve").Output(); err == nil {
			for _, ln := range strings.Split(string(out), "\n") {
				if i := strings.Index(ln, "REG_SZ"); i >= 0 {
					cands = append([]string{strings.TrimSpace(ln[i+6:])}, cands...)
				}
			}
		}
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func launchChrome(profileDir string) error {
	exe := chromeExe()
	if exe == "" {
		return errors.New("could not find chrome.exe to reopen Chrome")
	}
	args := []string{"--profile-directory=" + profileDir, "--restore-last-session"}
	if cfg, _ := loadConfigLoose(); !strings.EqualFold(cfg.UserDataDir, defaultUserDataDir()) {
		args = append(args, "--user-data-dir="+cfg.UserDataDir)
	}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | createNewProcessGroup}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

const regKey = `HKCU\Software\Google\Chrome\NativeMessagingHosts\` + hostName

func registerHost(manifestFile string, manifest []byte, udd string) error {
	out, err := hidden("reg", "add", regKey, "/ve", "/t", "REG_SZ", "/d", manifestFile, "/f").CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not register with Chrome: %v %s", err, out)
	}
	return nil
}

func unregisterHost(udd string) {
	hidden("reg", "delete", regKey, "/f").Run()
}

const uninstallKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\ProfileManagerHelper`

// registerUninstaller adds an entry to Settings > Apps so the helper can be removed like any app.
func registerUninstaller(exe string) {
	set := func(name, typ, val string) {
		hidden("reg", "add", uninstallKey, "/v", name, "/t", typ, "/d", val, "/f").Run()
	}
	set("DisplayName", "REG_SZ", "Profile Manager helper (for Chrome)")
	set("DisplayVersion", "REG_SZ", version)
	set("Publisher", "REG_SZ", "Profile Manager")
	set("DisplayIcon", "REG_SZ", exe)
	set("UninstallString", "REG_SZ", `"`+exe+`" --uninstall`)
	set("InstallLocation", "REG_SZ", filepath.Dir(exe))
	set("NoModify", "REG_DWORD", "1")
	set("NoRepair", "REG_DWORD", "1")
}

func unregisterUninstaller() { hidden("reg", "delete", uninstallKey, "/f").Run() }

// removeSelfLater deletes the install folder a few seconds after this program exits
// (Windows can't delete a running .exe).
func removeSelfLater(dir string) {
	c := exec.Command("cmd", "/c", `ping -n 4 127.0.0.1 >nul & rmdir /s /q "`+dir+`"`)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | detachedProcess}
	c.Start()
}

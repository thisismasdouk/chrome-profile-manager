// Profile Manager helper.
//
// Chrome extensions cannot touch files on disk, so this small program does the
// real work for the "Profile Manager" extension. Chrome starts it through
// "native messaging" whenever the extension needs something:
//
//   - list the profiles found in Chrome's "Local State" file
//   - work out which profile the extension is running in (that one is protected)
//   - schedule profiles for deletion
//
// Profiles can only be removed safely while Chrome is closed, so deletion is
// done by a background copy of this program (the "worker") that waits for
// Chrome to exit, removes the profiles, and can reopen Chrome afterwards.
//
// Run with no arguments (double-click) or with --install to register it with
// Chrome; --uninstall removes the registration.
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	version  = "1.0.0"
	hostName = "com.profile_manager.helper"
	// ID of the extension when loaded unpacked, fixed by the "key" in extension/manifest.json.
	extensionID = "cdjlealijjepaagcnlbbingaejlnfmon"
)

// ID the Chrome Web Store assigns after the first upload. Set it at build time:
//   go build -ldflags "-X main.storeExtensionID=<id>"
var storeExtensionID = ""

func allowedOrigins() []string {
	o := []string{"chrome-extension://" + extensionID + "/"}
	if storeExtensionID != "" {
		o = append(o, "chrome-extension://"+storeExtensionID+"/")
	}
	return o
}

func main() {
	args := os.Args[1:]
	switch {
	case len(args) == 0:
		runInstall("", true)
	case args[0] == "--install-quiet":
		// used by the Windows setup program: no prompts, error text on stdout
		if err := install(""); err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
		}
	case args[0] == "--install":
		udd := ""
		if len(args) > 1 {
			udd = args[1]
		}
		runInstall(udd, false)
	case args[0] == "--uninstall":
		runUninstall()
	case args[0] == "--worker":
		runWorker()
	case strings.HasPrefix(args[0], "chrome-extension://"):
		runHost(args[0])
	default:
		fmt.Println("Profile Manager helper", version)
		fmt.Println("Usage: profile-helper [--install [userDataDir] | --uninstall]")
		os.Exit(2)
	}
}

// ---------- native messaging ----------

func readMessage(r io.Reader) (map[string]any, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	if n > 64*1024*1024 {
		return nil, errors.New("message too large")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func writeMessage(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(w)
	if err := binary.Write(bw, binary.LittleEndian, uint32(len(b))); err != nil {
		return err
	}
	bw.Write(b)
	return bw.Flush()
}

func runHost(origin string) {
	extID := strings.TrimSuffix(strings.TrimPrefix(origin, "chrome-extension://"), "/")
	msg, err := readMessage(os.Stdin)
	if err != nil {
		writeMessage(os.Stdout, map[string]any{"ok": false, "error": "could not read request: " + err.Error()})
		return
	}
	resp, err := handle(extID, msg)
	if err != nil {
		resp = map[string]any{"ok": false, "error": err.Error()}
	} else {
		resp["ok"] = true
	}
	if err := writeMessage(os.Stdout, resp); err != nil {
		logf("write response: %v", err)
	}
}

func handle(extID string, msg map[string]any) (map[string]any, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	cmd, _ := msg["cmd"].(string)
	token, _ := msg["token"].(string)
	switch cmd {
	case "ping":
		return map[string]any{"version": version, "os": runtime.GOOS}, nil
	case "list":
		return cmdList(cfg, extID, token)
	case "sizes":
		return cmdSizes(cfg, stringList(msg["dirs"]))
	case "delete":
		quitNow, _ := msg["quitNow"].(bool)
		relaunch, _ := msg["relaunch"].(bool)
		return cmdDelete(cfg, extID, token, stringList(msg["dirs"]), quitNow, relaunch)
	case "cancel":
		os.Remove(jobPath())
		return map[string]any{}, nil
	case "clearResult":
		os.Remove(resultPath())
		return map[string]any{}, nil
	}
	return nil, fmt.Errorf("unknown command %q", cmd)
}

func stringList(v any) []string {
	arr, _ := v.([]any)
	out := []string{}
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// ---------- files next to the installed helper ----------

func appDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return filepath.Dir(exe)
}

func configPath() string { return filepath.Join(appDir(), "config.json") }
func jobPath() string    { return filepath.Join(appDir(), "pending-deletion.json") }
func alivePath() string  { return filepath.Join(appDir(), "worker.alive") }
func resultPath() string { return filepath.Join(appDir(), "last-result.json") }
func logPath() string    { return filepath.Join(appDir(), "helper.log") }

type Config struct {
	UserDataDir string `json:"userDataDir"`
}

func loadConfig() (*Config, error) {
	cfg := &Config{}
	if b, err := os.ReadFile(configPath()); err == nil {
		json.Unmarshal(b, cfg)
	}
	if cfg.UserDataDir == "" {
		cfg.UserDataDir = defaultUserDataDir()
	}
	if _, err := os.Stat(filepath.Join(cfg.UserDataDir, "Local State")); err != nil {
		return nil, fmt.Errorf("could not find Chrome's data folder at %s", cfg.UserDataDir)
	}
	return cfg, nil
}

func logf(format string, a ...any) {
	f, err := os.OpenFile(logPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", nowString(), fmt.Sprintf(format, a...))
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Job is the list of profiles waiting to be deleted once Chrome closes.
type Job struct {
	CreatedAt   string            `json:"createdAt"`
	UserDataDir string            `json:"userDataDir"`
	Dirs        []string          `json:"dirs"`
	Names       map[string]string `json:"names"`
	Protected   string            `json:"protected"` // profile in use; never deleted, reopened afterwards
	QuitNow     bool              `json:"quitNow"`
	Relaunch    bool              `json:"relaunch"`
}

type Result struct {
	At      string            `json:"at"`
	Deleted []string          `json:"deleted"`
	Failed  map[string]string `json:"failed"`
	Names   map[string]string `json:"names"`
}

func nowString() string { return time.Now().Format(time.RFC3339) }

func readJob() *Job {
	b, err := os.ReadFile(jobPath())
	if err != nil {
		return nil
	}
	var j Job
	if json.Unmarshal(b, &j) != nil || len(j.Dirs) == 0 {
		return nil
	}
	return &j
}

func writeJob(j *Job) error {
	b, _ := json.MarshalIndent(j, "", "  ")
	return writeFileAtomic(jobPath(), b)
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func workerAlive() bool {
	st, err := os.Stat(alivePath())
	return err == nil && time.Since(st.ModTime()) < 8*time.Second
}

// ensureWorker starts the background remover unless one is already running.
func ensureWorker() error {
	if workerAlive() {
		return nil
	}
	os.WriteFile(alivePath(), []byte(nowString()), 0o644)
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return spawnDetached(exe, "--worker")
}

// runWorker waits for Chrome to close, then deletes the profiles in the job.
func runWorker() {
	logf("worker started")
	give := time.Now().Add(12 * time.Hour)
	for {
		os.WriteFile(alivePath(), []byte(nowString()), 0o644)
		job := readJob()
		if job == nil {
			logf("worker: nothing pending (cancelled?), exiting")
			os.Remove(alivePath())
			return
		}
		if job.QuitNow {
			job.QuitNow = false
			writeJob(job)
			time.Sleep(1500 * time.Millisecond) // let the extension show its message
			logf("worker: asking Chrome to quit")
			quitChrome(job.UserDataDir)
		}
		if !chromeRunning(job.UserDataDir) {
			time.Sleep(2 * time.Second)
			if !chromeRunning(job.UserDataDir) {
				// re-read: the user may have cancelled at the last moment
				if job = readJob(); job != nil {
					performDeletion(job)
				}
				os.Remove(alivePath())
				return
			}
		}
		if time.Now().After(give) {
			logf("worker: giving up after 12h; will restart next time the extension opens")
			os.Remove(alivePath())
			return
		}
		time.Sleep(time.Second)
	}
}

func performDeletion(job *Job) {
	udd := job.UserDataDir
	res := Result{At: nowString(), Deleted: []string{}, Failed: map[string]string{}, Names: job.Names}

	ls, err := readLocalState(udd)
	if err != nil {
		logf("deletion aborted: %v", err)
		res.Failed["*"] = err.Error()
		saveResult(res)
		return
	}
	ic := infoCache(ls)
	prof, _ := ls["profile"].(map[string]any)
	// Profiles that had windows open when Chrome closed (to reopen them afterwards).
	wasOpen := []string{}
	if prof != nil {
		if arr, ok := prof["last_active_profiles"].([]any); ok {
			for _, v := range arr {
				wasOpen = append(wasOpen, str(v))
			}
		}
	}

	targets := []string{}
	for _, d := range job.Dirs {
		switch {
		case d == job.Protected || job.Protected == "":
			res.Failed[d] = "this is the profile you were using, so it was kept"
		case !validProfileDir(d):
			res.Failed[d] = "not a valid profile folder"
		default:
			if _, ok := ic[d]; !ok {
				// already gone from Chrome's list; still clean up a leftover folder
				if _, err := os.Stat(filepath.Join(udd, d, "Preferences")); err != nil {
					res.Failed[d] = "profile no longer exists"
					continue
				}
			}
			targets = append(targets, d)
		}
	}
	// Never leave Chrome with zero profiles.
	if len(ic) > 0 && countRemaining(ic, targets) == 0 {
		res.Failed["*"] = "at least one profile must remain"
		saveResult(res)
		os.Remove(jobPath())
		return
	}

	if len(targets) > 0 {
		// 1. Remove the profiles from Chrome's own list (with a backup first).
		orig, _ := os.ReadFile(filepath.Join(udd, "Local State"))
		os.WriteFile(filepath.Join(udd, "Local State.profile-manager-backup"), orig, 0o644)
		tset := map[string]bool{}
		for _, d := range targets {
			tset[d] = true
			delete(ic, d)
		}
		if prof != nil {
			for _, key := range []string{"last_active_profiles", "profiles_order"} {
				if arr, ok := prof[key].([]any); ok {
					kept := []any{}
					for _, v := range arr {
						if !tset[str(v)] {
							kept = append(kept, v)
						}
					}
					prof[key] = kept
				}
			}
			if tset[str(prof["last_used"])] {
				prof["last_used"] = job.Protected
			}
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(ls); err != nil {
			res.Failed["*"] = "could not update Chrome's profile list: " + err.Error()
			saveResult(res)
			return
		}
		if err := writeFileAtomic(filepath.Join(udd, "Local State"), bytes.TrimRight(buf.Bytes(), "\n")); err != nil {
			res.Failed["*"] = "could not update Chrome's profile list: " + err.Error()
			saveResult(res)
			return
		}

		// 2. Delete the profile folders.
		for _, d := range targets {
			if err := removeAllRetry(filepath.Join(udd, d)); err != nil {
				res.Failed[d] = "removed from Chrome, but some files could not be deleted: " + err.Error()
				logf("delete %s: %v", d, err)
			} else {
				res.Deleted = append(res.Deleted, d)
				logf("deleted profile %s (%s)", d, job.Names[d])
			}
		}
	}
	saveResult(res)
	os.Remove(jobPath())

	if job.Relaunch && job.Protected != "" {
		// Reopen the profile in use first, then every other profile that had
		// windows open, each with its tabs restored.
		deleted := map[string]bool{}
		for _, d := range res.Deleted {
			deleted[d] = true
		}
		order := []string{job.Protected}
		for _, d := range wasOpen {
			if d != job.Protected && !deleted[d] && validProfileDir(d) {
				order = append(order, d)
			}
		}
		for i, d := range order {
			if i > 0 {
				time.Sleep(3 * time.Second)
			}
			if err := launchChrome(d); err != nil {
				logf("relaunch %s: %v", d, err)
				break
			}
		}
	}
}

func countRemaining(ic map[string]any, targets []string) int {
	t := map[string]bool{}
	for _, d := range targets {
		t[d] = true
	}
	n := 0
	for d := range ic {
		if !t[d] && validProfileDir(d) {
			n++
		}
	}
	return n
}

func removeAllRetry(path string) error {
	var err error
	for i := 0; i < 5; i++ {
		if err = os.RemoveAll(path); err == nil {
			return nil
		}
		// read-only files block deletion on Windows; clear the flag and retry
		filepath.Walk(path, func(p string, info os.FileInfo, e error) error {
			if e == nil {
				os.Chmod(p, 0o777)
			}
			return nil
		})
		time.Sleep(time.Second)
	}
	return err
}

func saveResult(r Result) {
	b, _ := json.MarshalIndent(r, "", "  ")
	writeFileAtomic(resultPath(), b)
	logf("result: deleted=%v failed=%v", r.Deleted, r.Failed)
}

var _ = fmt.Sprintf

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Profile struct {
	Dir        string  `json:"dir"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	FullName   string  `json:"fullName"`
	LastUsed   float64 `json:"lastUsed"` // unix seconds, 0 if unknown
	Color      string  `json:"color"`
	Avatar     string  `json:"avatar,omitempty"` // data: URL
	IsCurrent  bool    `json:"isCurrent"`
	FolderSeen bool    `json:"folderExists"`
}

// readLocalState parses Chrome's "Local State" keeping numbers exactly as written.
func readLocalState(udd string) (map[string]any, error) {
	b, err := os.ReadFile(filepath.Join(udd, "Local State"))
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var ls map[string]any
	if err := dec.Decode(&ls); err != nil {
		return nil, fmt.Errorf("could not read Chrome's Local State file: %v", err)
	}
	return ls, nil
}

func infoCache(ls map[string]any) map[string]any {
	p, _ := ls["profile"].(map[string]any)
	if p == nil {
		return map[string]any{}
	}
	ic, _ := p["info_cache"].(map[string]any)
	if ic == nil {
		return map[string]any{}
	}
	return ic
}

func num(v any) float64 {
	switch x := v.(type) {
	case json.Number:
		f, _ := x.Float64()
		return f
	case float64:
		return x
	}
	return 0
}

func str(v any) string { s, _ := v.(string); return s }

// validProfileDir rejects anything that is not a plain folder name inside the
// Chrome data folder, or a special profile that must never be removed.
func validProfileDir(dir string) bool {
	if dir == "" || dir == "." || dir == ".." || strings.ContainsAny(dir, `/\:`) {
		return false
	}
	switch strings.ToLower(dir) {
	case "system profile", "guest profile":
		return false
	}
	return true
}

// findCurrentProfile finds the profile the extension is running in. The
// extension stores a random token in chrome.storage.local, which Chrome keeps
// in "<profile>/Local Extension Settings/<extension id>/". Only the profile
// the extension was opened from contains that exact token.
func findCurrentProfile(udd, extID, token string) string {
	if len(token) < 16 || extID == "" {
		return ""
	}
	needle := []byte(token)
	entries, err := os.ReadDir(udd)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(udd, e.Name(), "Local Extension Settings", extID)
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			n := f.Name()
			if !(strings.HasSuffix(n, ".log") || strings.HasSuffix(n, ".ldb")) {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, n))
			if err == nil && bytes.Contains(b, needle) {
				return e.Name()
			}
		}
	}
	return ""
}

func listProfiles(udd, current string) ([]Profile, error) {
	ls, err := readLocalState(udd)
	if err != nil {
		return nil, err
	}
	out := []Profile{}
	budget := 700 * 1024 // keep the reply under Chrome's 1 MB limit
	for dir, raw := range infoCache(ls) {
		info, _ := raw.(map[string]any)
		if info == nil || !validProfileDir(dir) {
			continue
		}
		p := Profile{
			Dir:       dir,
			Name:      str(info["name"]),
			Email:     str(info["user_name"]),
			FullName:  str(info["gaia_name"]),
			LastUsed:  num(info["active_time"]),
			IsCurrent: dir == current,
		}
		if p.Name == "" {
			p.Name = dir
		}
		if c, ok := info["default_avatar_fill_color"]; ok {
			p.Color = argbToCSS(int64(num(c)))
		} else if c, ok := info["profile_highlight_color"]; ok {
			p.Color = argbToCSS(int64(num(c)))
		}
		if st, err := os.Stat(filepath.Join(udd, dir)); err == nil && st.IsDir() {
			p.FolderSeen = true
			if a := avatarDataURL(filepath.Join(udd, dir, "Google Profile Picture.png")); a != "" && len(a) < budget {
				p.Avatar = a
				budget -= len(a)
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsCurrent != out[j].IsCurrent {
			return out[i].IsCurrent
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func argbToCSS(v int64) string {
	u := uint32(v)
	return fmt.Sprintf("#%02x%02x%02x", (u>>16)&0xff, (u>>8)&0xff, u&0xff)
}

// avatarDataURL shrinks the Google account picture to 64x64 and returns it as a data URL.
func avatarDataURL(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return ""
	}
	const size = 64
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// average a small block of source pixels for a smoother result
			x0 := b.Min.X + x*b.Dx()/size
			x1 := b.Min.X + (x+1)*b.Dx()/size
			y0 := b.Min.Y + y*b.Dy()/size
			y1 := b.Min.Y + (y+1)*b.Dy()/size
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if y1 <= y0 {
				y1 = y0 + 1
			}
			var r, g, bl, a, n uint32
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r, g, bl, a, n = r+cr, g+cg, bl+cb, a+ca, n+1
				}
			}
			dst.Set(x, y, color.RGBA64{uint16(r / n), uint16(g / n), uint16(bl / n), uint16(a / n)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// ---------- commands ----------

func cmdList(cfg *Config, extID, token string) (map[string]any, error) {
	current := findCurrentProfile(cfg.UserDataDir, extID, token)
	profiles, err := listProfiles(cfg.UserDataDir, current)
	if err != nil {
		return nil, err
	}
	resp := map[string]any{
		"version":     version,
		"userDataDir": cfg.UserDataDir,
		"current":     current,
		"profiles":    profiles,
	}
	if job := readJob(); job != nil {
		ensureWorker()
		resp["pending"] = job
	}
	if b, err := os.ReadFile(resultPath()); err == nil {
		var r any
		if json.Unmarshal(b, &r) == nil {
			resp["lastResult"] = r
		}
	}
	return resp, nil
}

func cmdSizes(cfg *Config, dirs []string) (map[string]any, error) {
	sizes := map[string]int64{}
	deadline := time.Now().Add(25 * time.Second)
	for _, d := range dirs {
		if !validProfileDir(d) || time.Now().After(deadline) {
			continue
		}
		var total int64
		filepath.WalkDir(filepath.Join(cfg.UserDataDir, d), func(_ string, e fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if time.Now().After(deadline) {
				return errors.New("timeout")
			}
			if !e.IsDir() {
				if info, err := e.Info(); err == nil {
					total += info.Size()
				}
			}
			return nil
		})
		sizes[d] = total
	}
	return map[string]any{"sizes": sizes}, nil
}

func cmdDelete(cfg *Config, extID, token string, dirs []string, quitNow, relaunch bool) (map[string]any, error) {
	current := findCurrentProfile(cfg.UserDataDir, extID, token)
	if current == "" {
		return nil, errors.New("could not tell which profile you are using right now, so nothing was deleted to keep it safe. Close this tab and open Profile Manager again")
	}
	if len(dirs) == 0 {
		return nil, errors.New("no profiles selected")
	}
	ls, err := readLocalState(cfg.UserDataDir)
	if err != nil {
		return nil, err
	}
	ic := infoCache(ls)
	for _, d := range dirs {
		if d == current {
			return nil, errors.New("the profile you are using right now cannot be deleted")
		}
		if !validProfileDir(d) {
			return nil, fmt.Errorf("%q is not a profile that can be deleted", d)
		}
		if _, ok := ic[d]; !ok {
			return nil, fmt.Errorf("%q is not in Chrome's profile list", d)
		}
	}
	job := readJob()
	if job == nil {
		job = &Job{CreatedAt: nowString()}
	}
	seen := map[string]bool{}
	for _, d := range job.Dirs {
		seen[d] = true
	}
	for _, d := range dirs {
		if !seen[d] {
			job.Dirs = append(job.Dirs, d)
			seen[d] = true
		}
	}
	for _, d := range dirs {
		info, _ := ic[d].(map[string]any)
		if job.Names == nil {
			job.Names = map[string]string{}
		}
		job.Names[d] = str(info["name"])
	}
	job.UserDataDir = cfg.UserDataDir
	job.Protected = current
	job.QuitNow = quitNow
	job.Relaunch = relaunch
	if err := writeJob(job); err != nil {
		return nil, err
	}
	os.Remove(resultPath())
	if err := ensureWorker(); err != nil {
		return nil, fmt.Errorf("could not start the background remover: %v", err)
	}
	return map[string]any{"pending": job}, nil
}

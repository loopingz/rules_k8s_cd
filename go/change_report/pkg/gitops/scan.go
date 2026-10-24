// Package gitops scans gitops output directories for manifests carrying the
// "# rules_k8s_cd:target <label>" header and maps each target to its files.
package gitops

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
)

// HeaderPrefix starts the first line of a gitops-exported manifest.
const HeaderPrefix = "# rules_k8s_cd:target "

// Result is the outcome of a scan.
type Result struct {
	// Targets maps a normalized "//pkg:name" label to its files, sorted.
	Targets map[string][]string
	// Unlabeled counts YAML files without a valid header.
	Unlabeled int
}

// Scan walks dirs for *.yaml / *.yml files and reads their header line. The
// identity -> label mapping is cached in cacheFile (empty disables caching).
// Files unmodified relative to the git index are identified by blob SHA, any
// other file by path|size|mtime, so a modified file never yields a stale label.
func Scan(dirs []string, cacheFile string) (Result, error) {
	cached := loadCache(cacheFile)
	res := Result{Targets: map[string][]string{}}
	dirty := false
	for _, dir := range dirs {
		blobs := gitBlobs(dir)
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil // symlinks and specials are not followed
			}
			ext := filepath.Ext(p)
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			var id string
			if sha, ok := blobs[filepath.ToSlash(rel)]; ok {
				id = "blob|" + sha
			} else {
				fi, err := d.Info()
				if err != nil {
					return err
				}
				id = fmt.Sprintf("stat|%s|%d|%d", p, fi.Size(), fi.ModTime().UnixNano())
			}
			lbl, ok := cached[id]
			if !ok {
				l, err := readLabel(p)
				if err != nil {
					return err
				}
				lbl = l
				cached[id] = lbl
				dirty = true
			}
			if lbl == "" {
				res.Unlabeled++
			} else {
				res.Targets[lbl] = append(res.Targets[lbl], p)
			}
			return nil
		})
		if err != nil {
			return Result{}, err
		}
	}
	for _, files := range res.Targets {
		sort.Strings(files)
	}
	if dirty && cacheFile != "" {
		if err := saveCache(cacheFile, cached); err != nil {
			return Result{}, err
		}
	}
	return res, nil
}

// readLabel returns the normalized label of the first line, or "" if the
// header is missing or its label does not parse.
func readLabel(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	line, err := bufio.NewReaderSize(f, 4096).ReadString('\n')
	if err != nil && line == "" {
		return "", nil // empty file (or read error with no data)
	}
	line = strings.TrimRight(line, "\r\n")
	rest, ok := strings.CutPrefix(line, HeaderPrefix)
	if !ok {
		return "", nil
	}
	l, perr := label.Parse(strings.TrimSpace(rest))
	if perr != nil {
		return "", nil
	}
	return l.String(), nil
}

// gitBlobs returns path (relative to dir, slash separated) -> blob SHA for the
// tracked files under dir that match the index. Modified, deleted or untracked
// files are absent, as is everything when dir is not in a git work tree.
func gitBlobs(dir string) map[string]string {
	out, err := exec.Command("git", "-C", dir, "ls-files", "-s", "-z").Output()
	if err != nil {
		return nil
	}
	mod, err := exec.Command("git", "-C", dir, "ls-files", "-m", "-z").Output()
	if err != nil {
		return nil
	}
	modified := map[string]bool{}
	for _, m := range bytes.Split(mod, []byte{0}) {
		modified[string(m)] = true
	}
	blobs := map[string]string{}
	conflict := map[string]bool{}
	for _, rec := range bytes.Split(out, []byte{0}) {
		meta, path, ok := strings.Cut(string(rec), "\t")
		if !ok {
			continue
		}
		f := strings.Fields(meta)
		if len(f) != 3 || f[2] != "0" || modified[path] || conflict[path] {
			delete(blobs, path)
			conflict[path] = true
			continue
		}
		blobs[path] = f[1]
	}
	return blobs
}

func loadCache(file string) map[string]string {
	m := map[string]string{}
	if file == "" {
		return m
	}
	b, err := os.ReadFile(file)
	if err != nil || json.Unmarshal(b, &m) != nil || m == nil {
		return map[string]string{}
	}
	return m
}

func saveCache(file string, m map[string]string) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".gitops-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}

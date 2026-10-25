// Package cache produces the dependency graph snapshot of a workspace (universe
// labels, per-target closures and the external dependency mapping), caching it
// on disk under a global key plus one key per package so that adding or
// removing a file only re-queries the package that owns it.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/extdeps"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/graph"
	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
)

// FormatVersion is the on-disk cache format; bump it on any change to the
// persisted data or to how it is computed.
const FormatVersion = 3

// Snapshot is the dependency graph of the universe at HEAD.
type Snapshot struct {
	Graph *graph.Graph
	// Universe holds the universe target labels in "//pkg:name" form, sorted.
	Universe []string
	// Closures is keyed by the Universe labels.
	Closures map[string]graph.Closure
	Mapping  extdeps.Mapping
	// ModuleBzlFiles are the workspace .bzl files used by MODULE.bazel and
	// *.MODULE.bazel (use_extension, use_repo_rule) plus their transitive
	// loads, sorted. They are .bzl files, so the GlobalKey covers them.
	ModuleBzlFiles []string
	PackageKeys    map[string]string
	GlobalKey      string
	// Tree maps every path at HEAD to its blob SHA; recomputed on each run.
	Tree map[string]string `json:"-"`
}

// Packages returns the workspace packages ("" for the root), sorted.
func (s *Snapshot) Packages() []string {
	out := make([]string, 0, len(s.PackageKeys))
	for p := range s.PackageKeys {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Stats describes how a Snapshot was obtained.
type Stats struct {
	Mode              string // "full", "partial" or "hit"
	RequeriedPackages []string
}

// Options configures Load.
type Options struct {
	// Workspace is the workspace root (also Repo.Dir); BUILD and .bzl files
	// are read from its working tree, which is what Bazel evaluates.
	Workspace string
	// CacheDir holds the cache files; empty disables caching.
	CacheDir string
	// Universe is the bazel query expression of the targets to report on.
	Universe string
	Runner   bazelq.Runner
	Repo     gitx.Repo
	// Warn receives non-fatal problems; nil ignores them.
	Warn func(string)
}

// persisted is the on-disk cache format.
type persisted struct {
	FormatVersion int
	Packages      []string
	// MappedFiles are the Snapshot.Mapping.FileRepos keys; InputsDigest
	// hashes the working-tree content of inputPaths(tree, MappedFiles).
	MappedFiles  []string
	InputsDigest string
	Snapshot     Snapshot
}

// Load returns the snapshot for HEAD, from the cache when it is still valid.
// A missing or doubtful cache, a global key change or a new/removed package
// triggers a full query; packages whose key changed are re-queried alone.
// Only a failing full query is an error; cache I/O problems are warnings.
func Load(o Options) (*Snapshot, Stats, error) {
	warn := o.Warn
	if warn == nil {
		warn = func(string) {}
	}
	tree, err := o.Repo.TreeFiles("HEAD")
	if err != nil {
		return nil, Stats{}, fmt.Errorf("listing HEAD tree: %w", err)
	}
	gk := GlobalKey(o.Universe, bazelVersion(o.Workspace), tree)
	pkgs := Packages(tree)
	keys := packageKeys(tree, pkgs)

	file, err := cachePath(o)
	if err != nil {
		warn(fmt.Sprintf("graph cache disabled: %v", err))
	}
	if file != "" {
		if c := readCache(file, gk, warn); c != nil && c.InputsDigest == inputsDigest(o.Workspace, inputPaths(tree, c.MappedFiles)) {
			if s, st, ok := incremental(o, c, tree, pkgs, keys, warn); ok {
				if st.Mode != "hit" {
					save(file, o.Workspace, s, pkgs, warn)
				}
				return s, st, nil
			}
		}
	}

	s, err := full(o, tree, pkgs, keys, gk, warn)
	if err != nil {
		return nil, Stats{}, err
	}
	if file != "" {
		save(file, o.Workspace, s, pkgs, warn)
	}
	return s, Stats{Mode: "full"}, nil
}

// bazelVersion identifies the Bazel version without running Bazel (a cache
// hit must not call the Runner): the working-tree .bazelversion plus the
// USE_BAZEL_VERSION override honoured by bazelisk.
func bazelVersion(ws string) string {
	v, _ := os.ReadFile(filepath.Join(ws, ".bazelversion"))
	return strings.TrimSpace(string(v)) + ";" + os.Getenv("USE_BAZEL_VERSION")
}

func cachePath(o Options) (string, error) {
	if o.CacheDir == "" {
		return "", nil
	}
	abs, err := filepath.Abs(o.Workspace)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join(o.CacheDir, hex.EncodeToString(sum[:])[:16]+".json"), nil
}

// readCache returns the cached data if it is readable and matches the
// current format and global key, nil otherwise.
func readCache(file, gk string, warn func(string)) *persisted {
	data, err := os.ReadFile(file)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			warn(fmt.Sprintf("graph cache %s unreadable, rebuilding: %v", file, err))
		}
		return nil
	}
	var c persisted
	if err := json.Unmarshal(data, &c); err != nil {
		warn(fmt.Sprintf("graph cache %s corrupt, rebuilding: %v", file, err))
		return nil
	}
	s := &c.Snapshot
	if c.FormatVersion != FormatVersion || s.GlobalKey != gk || s.Graph == nil || s.Graph.Nodes == nil || s.Closures == nil || s.PackageKeys == nil {
		return nil
	}
	if s.Graph.Loads == nil {
		s.Graph.Loads = map[string][]string{}
	}
	if s.Mapping.FileRepos == nil {
		s.Mapping.FileRepos = map[string][]string{}
	}
	return &c
}

// save writes the snapshot atomically; failures are only warnings.
func save(file, ws string, s *Snapshot, pkgs map[string]bool, warn func(string)) {
	err := func() error {
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		mapped := make([]string, 0, len(s.Mapping.FileRepos))
		for p := range s.Mapping.FileRepos {
			mapped = append(mapped, p)
		}
		sort.Strings(mapped)
		data, err := json.Marshal(persisted{
			FormatVersion: FormatVersion,
			Packages:      sortedKeys(pkgs),
			MappedFiles:   mapped,
			InputsDigest:  inputsDigest(ws, inputPaths(s.Tree, mapped)),
			Snapshot:      *s,
		})
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp(dir, filepath.Base(file)+".tmp-*")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name()) // no-op after a successful rename
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			return err
		}
		if err := tmp.Close(); err != nil {
			return err
		}
		return os.Rename(tmp.Name(), file)
	}()
	if err != nil {
		warn(fmt.Sprintf("saving graph cache %s: %v", file, err))
	}
}

// incremental reuses the cache c. ok is false when a full query is needed.
func incremental(o Options, c *persisted, tree map[string]string, pkgs map[string]bool, keys map[string]string, warn func(string)) (s *Snapshot, st Stats, ok bool) {
	if !sameSet(c.Packages, pkgs) {
		return nil, Stats{}, false // new or removed package
	}
	var changed []string
	for p := range pkgs {
		if old, found := c.Snapshot.PackageKeys[p]; !found || old != keys[p] {
			changed = append(changed, p)
		}
	}
	sort.Strings(changed)
	s = &c.Snapshot
	s.Tree = tree
	if len(changed) == 0 {
		return s, Stats{Mode: "hit"}, true
	}

	fallback := func(err error) (*Snapshot, Stats, bool) {
		warn(fmt.Sprintf("partial graph update failed, re-querying everything: %v", err))
		return nil, Stats{}, false
	}
	universe, err := universeLabels(o.Runner, o.Universe)
	if err != nil {
		return fallback(err)
	}
	if !equalStrings(universe, s.Universe) {
		return nil, Stats{}, false
	}
	pats := make([]string, len(changed))
	for i, p := range changed {
		pats[i] = "//" + p + ":*" // all targets, including files referenced by other packages
	}
	nodes, err := bazelq.Query(o.Runner, "deps(set("+strings.Join(pats, " ")+"))")
	if err != nil {
		return fallback(err)
	}
	s.Graph.ReplacePackages(changed, nodes)
	mainNames := moduleNames(o.Workspace)
	for _, p := range changed {
		for _, name := range []string{"BUILD.bazel", "BUILD"} {
			bf := path.Join(p, name)
			delete(s.Graph.Loads, bf)
			if _, inTree := tree[bf]; inTree {
				setLoads(s.Graph, o.Workspace, bf, p, mainNames)
			}
		}
	}
	changedSet := make(map[string]bool, len(changed))
	for _, p := range changed {
		changedSet[p] = true
	}
	bf := buildFileFunc(tree)
	for _, t := range s.Universe {
		if cl, found := s.Closures[t]; found && !intersects(cl.Packages, changedSet) {
			continue
		}
		s.Closures[t] = s.Graph.Closure(t, bf)
	}
	s.PackageKeys = keys
	return s, Stats{Mode: "partial", RequeriedPackages: changed}, true
}

// full queries the whole universe.
func full(o Options, tree map[string]string, pkgs map[string]bool, keys map[string]string, gk string, warn func(string)) (*Snapshot, error) {
	universe, err := universeLabels(o.Runner, o.Universe)
	if err != nil {
		return nil, fmt.Errorf("querying universe: %w", err)
	}
	nodes, err := bazelq.Query(o.Runner, "deps("+o.Universe+")")
	if err != nil {
		return nil, fmt.Errorf("querying dependency graph: %w", err)
	}
	g := graph.New(nodes)
	mainNames := moduleNames(o.Workspace)
	for p := range tree {
		if isBuildFile(p) || strings.HasSuffix(p, ".bzl") {
			setLoads(g, o.Workspace, p, filePackage(p, pkgs), mainNames)
		}
	}

	mapping := extdeps.Mapping{FileRepos: map[string][]string{}}
	if mod, err := os.ReadFile(filepath.Join(o.Workspace, "MODULE.bazel")); err == nil {
		lock, _ := os.ReadFile(filepath.Join(o.Workspace, "MODULE.bazel.lock")) // empty → extdeps generates it
		m, warns, err := extdeps.Load(o.Workspace, o.Runner, lock, mod)
		if err != nil {
			return nil, fmt.Errorf("loading external dependency mapping: %w", err)
		}
		for _, id := range warns {
			warn(fmt.Sprintf("show_extension failed for %s; files it reads are not mapped", id))
		}
		mapping = m
	}

	var moduleBzl []string
	for p := range tree {
		if p == "MODULE.bazel" || strings.HasSuffix(p, ".MODULE.bazel") {
			if src, err := os.ReadFile(filepath.Join(o.Workspace, filepath.FromSlash(p))); err == nil {
				moduleBzl = append(moduleBzl, extdeps.ModuleBzlPaths(src, mainNames)...)
			}
		}
	}

	bf := buildFileFunc(tree)
	closures := make(map[string]graph.Closure, len(universe))
	for _, t := range universe {
		closures[t] = g.Closure(t, bf)
	}
	return &Snapshot{
		Graph:          g,
		Universe:       universe,
		Closures:       closures,
		Mapping:        mapping,
		ModuleBzlFiles: g.LoadClosure(moduleBzl),
		PackageKeys:    keys,
		GlobalKey:      gk,
		Tree:           tree,
	}, nil
}

// universeLabels returns the universe in sorted "//pkg:name" form.
func universeLabels(r bazelq.Runner, expr string) ([]string, error) {
	raw, err := bazelq.Labels(r, expr)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(raw))
	for _, x := range raw {
		if l, err := label.Parse(x); err == nil {
			x = l.String()
		}
		out = append(out, x)
	}
	sort.Strings(out)
	return out, nil
}

// setLoads records the loads of workspace file p of package pkg, read from
// the working tree. A file missing from the working tree is not evaluated by
// Bazel and is skipped.
func setLoads(g *graph.Graph, ws, p, pkg string, mainNames []string) {
	src, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(p)))
	if err != nil {
		return
	}
	g.SetLoads(p, graph.ParseLoads(src, pkg, mainNames))
}

// filePackage returns the package of workspace file p: its nearest enclosing
// package, else its directory (Bazel cannot load such a file anyway).
func filePackage(p string, pkgs map[string]bool) string {
	if o, ok := owner(p, pkgs); ok {
		return o
	}
	return pkgDir(p)
}

// moduleNames returns the names the root module may use for itself in labels
// (module name and repo_name from MODULE.bazel).
func moduleNames(ws string) []string {
	mod, err := os.ReadFile(filepath.Join(ws, "MODULE.bazel"))
	if err != nil {
		return nil
	}
	return extdeps.MainRepoNames(mod)
}

// buildFileFunc returns the BUILD file path of a package: BUILD.bazel if it
// is in tree, BUILD otherwise.
func buildFileFunc(tree map[string]string) func(string) string {
	return func(pkg string) string {
		if bf := path.Join(pkg, "BUILD.bazel"); tree[bf] != "" {
			return bf
		}
		return path.Join(pkg, "BUILD")
	}
}

func sameSet(list []string, set map[string]bool) bool {
	if len(list) != len(set) {
		return false
	}
	for _, x := range list {
		if !set[x] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func intersects(list []string, set map[string]bool) bool {
	for _, x := range list {
		if set[x] {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Command change_report reports which Bazel targets changed since a baseline
// commit, the commits responsible, and removed gitops targets.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitops"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
	"loopingz.com/rules_k8s_cd/change_report/pkg/render"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
)

type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

// ignorePathFunc returns an IgnorePath hook matching any of globs for every
// label, or nil when there are none.
func ignorePathFunc(globs []string) func(label, path string) bool {
	if len(globs) == 0 {
		return nil
	}
	return func(_, p string) bool {
		for _, g := range globs {
			if report.MatchGlob(g, p) {
				return true
			}
		}
		return false
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}

func run(args []string, stdout, stderr io.Writer, env func(string) string) int {
	fs := flag.NewFlagSet("change_report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		since       = fs.String("since", "", "baseline commit for all targets; empty reports every target as changed")
		sinceFile   = fs.String("since-file", "", "JSON file {\"//pkg:target\": \"<sha>\"} of per-target baselines")
		targets     = fs.String("targets", "", "bazel query expression of the universe (required)")
		format      = fs.String("format", "targets", "output format: targets|json|markdown|removed")
		output      = fs.String("output", "", "output file (default stdout)")
		commitURL   = fs.String("commit-url", "", "commit URL template with {sha}; default derived from the origin remote")
		cacheDir    = fs.String("cache-dir", "", "cache directory (default $XDG_CACHE_HOME/rules_k8s_cd/change_report)")
		workspace   = fs.String("workspace", "", "workspace directory (default $BUILD_WORKSPACE_DIRECTORY, else cwd)")
		bazel       = fs.String("bazel", "bazel", "bazel binary")
		debugCache  = fs.Bool("debug-cache", false, "print the graph cache mode to stderr")
		gitopsDirs  stringSlice
		globalFiles stringSlice
		ignores     stringSlice
	)
	fs.Var(&gitopsDirs, "gitops-dir", "directory scanned for gitops manifests (repeatable)")
	fs.Var(&globalFiles, "global-file", "extra glob of files affecting every target (repeatable)")
	fs.Var(&ignores, "ignore", "glob of files never attributed to any target (repeatable)")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *targets == "" {
		fmt.Fprintln(stderr, "error: --targets is required")
		fs.Usage()
		return 2
	}
	switch *format {
	case "targets", "json", "markdown", "removed":
	default:
		fmt.Fprintf(stderr, "error: invalid --format %q (want targets|json|markdown|removed)\n", *format)
		fs.Usage()
		return 2
	}

	fail := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "error: "+format+"\n", a...)
		return 1
	}
	warn := func(msg string) { fmt.Fprintf(stderr, "warning: %s\n", msg) }

	// Under `bazel run` the process cwd is the runfiles tree; relative paths
	// given by the user are relative to BUILD_WORKING_DIRECTORY.
	resolve := func(p string) string { return resolvePath(env, p) }

	var perTarget map[string]string
	if *sinceFile != "" {
		data, err := os.ReadFile(resolve(*sinceFile))
		if err != nil {
			return fail("--since-file: %v", err)
		}
		if err := json.Unmarshal(data, &perTarget); err != nil {
			return fail("--since-file: %v", err)
		}
	}

	ws := *workspace
	if ws != "" {
		ws = resolve(ws)
	} else {
		ws = env("BUILD_WORKSPACE_DIRECTORY")
	}
	if ws == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fail("%v", err)
		}
		ws = wd
	}
	ws, err := filepath.Abs(ws)
	if err != nil {
		return fail("%v", err)
	}
	repo := gitx.Repo{Dir: ws}
	head, err := repo.RevParse("HEAD")
	if err != nil {
		return fail("resolving HEAD: %v", err)
	}

	cd := *cacheDir
	if cd != "" {
		cd = resolve(cd)
	} else {
		if x := env("XDG_CACHE_HOME"); x != "" {
			cd = filepath.Join(x, "rules_k8s_cd", "change_report")
		} else if uc, err := os.UserCacheDir(); err == nil {
			cd = filepath.Join(uc, "rules_k8s_cd", "change_report")
		}
	}
	loadCacheDir := cd
	if dirty, err := repo.Dirty(); err != nil {
		warn(fmt.Sprintf("cannot check working tree state: %v; graph cache disabled for this run", err))
		loadCacheDir = ""
	} else if dirty {
		warn("working tree has uncommitted changes; graph cache disabled for this run")
		loadCacheDir = ""
	}

	runner := bazelq.ExecRunner{Bazel: *bazel, Dir: ws}
	snap, stats, err := cache.Load(cache.Options{
		Workspace: ws,
		CacheDir:  loadCacheDir,
		Universe:  *targets,
		Runner:    runner,
		Repo:      repo,
		Warn:      warn,
	})
	if err != nil {
		return fail("%v", err)
	}
	if *debugCache {
		fmt.Fprintf(stderr, "cache: %s\n", stats.Mode)
	}

	if _, err := os.Stat(filepath.Join(ws, "MODULE.bazel.lock")); err == nil {
		if tracked, err := repo.Tracked("MODULE.bazel.lock"); err != nil {
			warn(fmt.Sprintf("cannot check whether MODULE.bazel.lock is committed: %v", err))
		} else if !tracked {
			warn("MODULE.bazel.lock is not committed; dependency changes use extension-level precision only")
		}
	}

	baselines, bw := report.ResolveBaselines(repo, head, snap.Universe, *since, perTarget)
	for _, w := range bw {
		warn(w)
	}
	commits, inRange, err := report.CollectHistory(repo, head, baselines)
	if err != nil {
		return fail("%v", err)
	}

	var gres *gitops.Result
	if len(gitopsDirs) > 0 {
		abs := make([]string, len(gitopsDirs))
		for i, d := range gitopsDirs {
			a, err := filepath.Abs(resolve(d))
			if err != nil {
				return fail("%v", err)
			}
			abs[i] = a
		}
		cacheFile := ""
		if cd != "" {
			sum := sha256.Sum256([]byte(strings.Join(abs, "\n")))
			cacheFile = filepath.Join(cd, "gitops-"+hex.EncodeToString(sum[:])[:16]+".json")
		}
		res, err := gitops.Scan(abs, cacheFile)
		if err != nil {
			return fail("scanning gitops dirs: %v", err)
		}
		if res.Unlabeled > 0 {
			warn(fmt.Sprintf("%d gitops file(s) without a rules_k8s_cd:target header were ignored", res.Unlabeled))
		}
		gres = &res
	}
	existing := existingOutsideUniverse(runner, gres, snap.Universe, warn)

	fileAt, tracked := report.GitAccessors(repo, warn)
	rep := report.Build(report.Input{
		Head:           head,
		Baselines:      baselines,
		Universe:       snap.Universe,
		Closures:       snap.Closures,
		Mapping:        snap.Mapping,
		Commits:        commits,
		InRange:        inRange,
		FileAt:         fileAt,
		Tracked:        tracked,
		GlobalGlobs:    globalFiles,
		IgnorePath:     ignorePathFunc(ignores),
		ModuleBzlFiles: snap.ModuleBzlFiles,
		Gitops:         gres,
		Existing:       existing,
	})
	for _, w := range rep.Warnings {
		warn(w)
	}
	// Removed files are shown relative to the user's working directory.
	wd := workingDir(env)
	for i := range rep.Targets {
		if rep.Targets[i].Status == report.Removed {
			for j, f := range rep.Targets[i].Files {
				rep.Targets[i].Files[j] = displayPath(wd, f)
			}
			sort.Strings(rep.Targets[i].Files)
		}
	}

	url := *commitURL
	if url == "" {
		url = repo.CommitURLTemplate()
	}
	var buf bytes.Buffer
	switch *format {
	case "targets":
		err = render.Targets(&buf, rep)
	case "removed":
		err = render.Removed(&buf, rep)
	case "json":
		err = render.JSON(&buf, rep, url)
	case "markdown":
		err = render.Markdown(&buf, rep, url)
	}
	if err != nil {
		return fail("%v", err)
	}
	if *output != "" {
		if err := writeFileAtomic(resolve(*output), buf.Bytes()); err != nil {
			return fail("%v", err)
		}
	} else if _, err := stdout.Write(buf.Bytes()); err != nil {
		return fail("%v", err)
	}
	return 0
}

// existingOutsideUniverse returns the gitops header labels outside universe
// that must not be marked removed: those `bazel query` still finds, or all of
// them if the query fails (a destructive output resolves doubt to "keep").
func existingOutsideUniverse(r bazelq.Runner, gres *gitops.Result, universe []string, warn func(string)) map[string]bool {
	if gres == nil {
		return nil
	}
	in := make(map[string]bool, len(universe))
	for _, l := range universe {
		in[l] = true
	}
	var candidates []string
	for l := range gres.Targets {
		if !in[l] {
			candidates = append(candidates, l)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Strings(candidates)
	existing := map[string]bool{}
	found, err := bazelq.ExistingLabels(r, candidates)
	if err != nil {
		warn(fmt.Sprintf("cannot check whether %d gitops target(s) outside --targets still exist, none marked removed: %v", len(candidates), err))
		for _, l := range candidates {
			existing[l] = true
		}
		return existing
	}
	for _, f := range found {
		if l, err := label.Parse(f); err == nil {
			f = l.String()
		}
		existing[f] = true
	}
	n := 0
	for _, l := range candidates {
		if existing[l] {
			n++
		}
	}
	if n > 0 {
		warn(fmt.Sprintf("%d gitops target(s) outside --targets still exist and were not marked removed", n))
	}
	return existing
}

// workingDir is the user's working directory: BUILD_WORKING_DIRECTORY under
// `bazel run`, else the process cwd ("" if unknown).
func workingDir(env func(string) string) string {
	if wd := env("BUILD_WORKING_DIRECTORY"); wd != "" {
		return wd
	}
	wd, _ := os.Getwd()
	return wd
}

// displayPath returns absolute path p relative to base when p is under it,
// else p unchanged.
func displayPath(base, p string) string {
	if base == "" {
		return p
	}
	rel, err := filepath.Rel(base, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return p
	}
	return rel
}

// resolvePath makes a relative path absolute against BUILD_WORKING_DIRECTORY
// (set by `bazel run`), else leaves it for the process cwd.
func resolvePath(env func(string) string, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	if wd := env("BUILD_WORKING_DIRECTORY"); wd != "" {
		return filepath.Join(wd, p)
	}
	return p
}

// writeFileAtomic writes data to a temp file next to path and renames it into
// place, so a failure never leaves path truncated.
func writeFileAtomic(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".change_report-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, 0o644)
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

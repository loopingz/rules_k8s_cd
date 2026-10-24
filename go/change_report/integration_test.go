//go:build integration

// End-to-end test of the change_report CLI against a real Bazel.
//
// Run with:
//
//	cd go && go test -tags integration ./change_report/ -run Integration -v -timeout 30m
//
// It needs `bazel` on PATH (and network access or a warm repository cache for
// Bazel's own module dependencies). The Bazel target
// //go/change_report:integration_test is tagged manual and is not expected to
// pass under `bazel test`: running Bazel inside the Bazel sandbox is not
// supported here.
//
// The workspace template lives in testdata/workspace. Files there that Bazel
// would interpret in this repository (BUILD.bazel, MODULE.bazel) carry a
// ".tmpl" suffix, stripped on copy; gitignore.tmpl becomes .gitignore.
//
// The template's .gitignore lists MODULE.bazel.lock: Bazel writes the lockfile
// on every query, and an untracked one would make the tree dirty and disable
// the graph cache. The extension subtest later force-adds the lockfile to
// exercise the Level 2 (lockfile diff) path as well.
package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/extdeps"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
)

const itUniverse = `filter("gitops\.", //...)`

var (
	itProd   = "//app:gitops.prod"
	itDev    = "//other:gitops.dev"
	itExtDev = "//app2:gitops.dev"
	itGen    = "//gen:gitops.gen"
	itAll    = []string{itExtDev, itProd, itGen, itDev} // sorted
)

type itEnv struct {
	t        *testing.T
	root     string // git toplevel
	ws       string // Bazel workspace, a subdirectory of root
	cacheDir string
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in -short mode")
	}
	if _, err := exec.LookPath("bazel"); err != nil {
		t.Skip("bazel not on PATH")
	}

	// Isolate git from the user's configuration; run() inherits this env.
	home := t.TempDir()
	gitconfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gitconfig, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_CONFIG_GLOBAL":   gitconfig,
		"GIT_AUTHOR_NAME":     "t",
		"GIT_AUTHOR_EMAIL":    "t@t",
		"GIT_COMMITTER_NAME":  "t",
		"GIT_COMMITTER_EMAIL": "t@t",
	} {
		t.Setenv(k, v)
	}

	// The workspace is a subdirectory of the git repository (monorepo layout),
	// so every subtest also exercises workspace-relative git paths.
	root := t.TempDir()
	e := &itEnv{t: t, root: root, ws: filepath.Join(root, "ws"), cacheDir: t.TempDir()}
	copyTemplate(t, filepath.Join("testdata", "workspace"), e.ws)
	// Pin the workspace to this repository's Bazel version (bazelisk).
	if v, err := os.ReadFile(filepath.Join("..", "..", ".bazelversion")); err == nil {
		e.write(".bazelversion", string(v))
	}
	t.Cleanup(func() {
		// Stop the workspace's server and free its output base.
		cmd := exec.Command("bazel", "clean", "--expunge")
		cmd.Dir = e.ws
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("bazel clean --expunge: %v\n%s", err, out)
		}
	})
	e.git("-C", e.root, "init", "-q")
	if err := os.WriteFile(filepath.Join(e.root, "README.md"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.commit("initial")

	// Warm up: every target is changed without a baseline; the cache is built.
	out, errOut := e.run(0, "--debug-cache")
	if want := lines(itAll...); out != want {
		t.Fatalf("initial run: got %q, want %q (stderr %s)", out, want, errOut)
	}
	if !strings.Contains(errOut, "cache: full") {
		t.Errorf("initial run: want cache: full, stderr %s", errOut)
	}

	t.Run("ChangeOutsideWorkspace", func(t *testing.T) {
		prev := e.head()
		if err := os.WriteFile(filepath.Join(e.root, "README.md"), []byte("changed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		e.commit("edit file outside the workspace")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		if out != "" {
			t.Errorf("got %q, want nothing (stderr %s)", out, errOut)
		}
		if !strings.Contains(errOut, "cache: hit") {
			t.Errorf("want cache: hit, stderr %s", errOut)
		}
	})

	t.Run("EditFile", func(t *testing.T) {
		prev := e.head()
		e.write("pkg/manifests/a.yaml", "a: 2\n")
		e.commit("edit a.yaml")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		if want := lines(itProd); out != want {
			t.Errorf("got %q, want %q (stderr %s)", out, want, errOut)
		}
		// File content edits do not change any package key.
		if !strings.Contains(errOut, "cache: hit") {
			t.Errorf("want cache: hit, stderr %s", errOut)
		}
	})

	t.Run("AddFileUnderMacroGlob", func(t *testing.T) {
		prev := e.head()
		e.write("pkg/manifests/c.yaml", "c: 1\n")
		sha := e.commit("add c.yaml")
		out, errOut := e.run(0, "--since", prev, "--format", "json", "--debug-cache")
		if !strings.Contains(errOut, "cache: partial") {
			t.Errorf("want cache: partial, stderr %s", errOut)
		}
		rep := parseJSON(t, out)
		got := rep.changed()
		if want := []string{itProd}; !reflect.DeepEqual(got, want) {
			t.Fatalf("changed = %v, want %v (stderr %s)", got, want, errOut)
		}
		tg := rep.target(itProd)
		if !reflect.DeepEqual(tg.Commits, []string{sha}) {
			t.Errorf("commits = %v, want [%s]", tg.Commits, sha)
		}
		if !slices.Contains(tg.Files, "pkg/manifests/c.yaml") {
			t.Errorf("files = %v, want pkg/manifests/c.yaml", tg.Files)
		}
		if c, ok := rep.Commits[sha]; !ok || c.Subject != "add c.yaml" {
			t.Errorf("commits[%s] = %+v, want subject %q", sha, c, "add c.yaml")
		}
		// The partially updated, persisted graph must include the new file.
		snap, st := e.load(t, e.cacheDir)
		if st.Mode != "hit" {
			t.Errorf("reloading the saved snapshot: mode %q, want hit", st.Mode)
		}
		if !slices.Contains(snap.Closures[itProd].Files, "pkg/manifests/c.yaml") {
			t.Errorf("closure of %s = %v, want pkg/manifests/c.yaml", itProd, snap.Closures[itProd].Files)
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		prev := e.head()
		e.remove("pkg/manifests/b.yaml")
		e.commit("delete b.yaml")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		if want := lines(itProd); out != want {
			t.Errorf("got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "cache: partial") {
			t.Errorf("want cache: partial, stderr %s", errOut)
		}
	})

	t.Run("CacheHit", func(t *testing.T) {
		prev := e.head()
		for i := 0; i < 2; i++ {
			out, errOut := e.run(0, "--since", prev, "--debug-cache")
			if out != "" {
				t.Errorf("run %d: got %q, want nothing", i, out)
			}
			if i == 1 && !strings.Contains(errOut, "cache: hit") {
				t.Errorf("second run: want cache: hit, stderr %s", errOut)
			}
		}
	})

	t.Run("GeneratedFileAcrossPartialRequery", func(t *testing.T) {
		prev := e.head()
		e.write("gen/extra.txt", "x\n")
		e.commit("add unrelated file to gen")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		// An added file may change a glob of its package: conservative.
		if want := lines(itGen); out != want {
			t.Errorf("add: got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "cache: partial") {
			t.Errorf("add: want cache: partial, stderr %s", errOut)
		}
		// The partially re-queried graph must still reach src.txt through
		// the generated file's generating rule.
		prev = e.head()
		e.write("gen/src.txt", "g: 2\n")
		e.commit("edit genrule source")
		out, errOut = e.run(0, "--since", prev, "--debug-cache")
		if want := lines(itGen); out != want {
			t.Errorf("edit: got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "cache: hit") {
			t.Errorf("edit: want cache: hit, stderr %s", errOut)
		}
	})

	t.Run("EditMacroBzl", func(t *testing.T) {
		prev := e.head()
		e.write("pkg/m.bzl", e.read("pkg/m.bzl")+"# comment\n")
		e.commit("edit macro")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		if want := lines(itProd); out != want {
			t.Errorf("got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "cache: full") {
			t.Errorf("want cache: full, stderr %s", errOut)
		}
	})

	t.Run("EditBuildFile", func(t *testing.T) {
		prev := e.head()
		e.write("other/BUILD.bazel", e.read("other/BUILD.bazel")+"# comment\n")
		e.commit("edit other BUILD")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		if want := lines(itDev); out != want {
			t.Errorf("got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "cache: partial") {
			t.Errorf("want cache: partial, stderr %s", errOut)
		}
	})

	t.Run("DirtyTree", func(t *testing.T) {
		e.write("other/o.yaml", "o: dirty\n")
		out, errOut := e.run(0, "--since", e.head(), "--debug-cache")
		if out != "" {
			t.Errorf("got %q, want nothing (uncommitted changes are not reported)", out)
		}
		if !strings.Contains(errOut, "working tree has uncommitted changes") || !strings.Contains(errOut, "cache: full") {
			t.Errorf("want dirty warning and cache: full, stderr %s", errOut)
		}
		e.git("checkout", "--", "other/o.yaml")
	})

	t.Run("ModuleBazelChange", func(t *testing.T) {
		prev := e.head()
		e.write("MODULE.bazel", e.read("MODULE.bazel")+"# a comment\n")
		e.commit("comment MODULE.bazel")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		if want := lines(itAll...); out != want {
			t.Errorf("got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "cache: full") {
			t.Errorf("want cache: full, stderr %s", errOut)
		}
		if !strings.Contains(errOut, "MODULE.bazel.lock is not committed") {
			t.Errorf("want untracked lockfile warning, stderr %s", errOut)
		}
	})

	t.Run("RemovedTarget", func(t *testing.T) {
		e.write("deploy/gone.yaml", "# rules_k8s_cd:target //gone:gitops.prod\nkind: ConfigMap\n")
		e.write("deploy/prod.yaml", "# rules_k8s_cd:target //app:gitops.prod\nkind: ConfigMap\n")
		e.write("deploy/plain.yaml", "kind: ConfigMap\n")
		e.commit("add gitops dir")
		dir := filepath.Join(e.ws, "deploy")
		out, errOut := e.run(0, "--since", e.head(), "--format", "removed", "--gitops-dir", dir)
		if want := lines(filepath.Join(dir, "gone.yaml")); out != want {
			t.Errorf("got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "1 gitops file(s) without a rules_k8s_cd:target header") {
			t.Errorf("want unlabeled warning, stderr %s", errOut)
		}
		out, errOut = e.run(0, "--since", e.head(), "--format", "json", "--gitops-dir", dir)
		if tg := parseJSON(t, out).target("//gone:gitops.prod"); tg == nil || tg.Status != "removed" {
			t.Errorf("//gone:gitops.prod = %+v, want status removed (stderr %s)", tg, errOut)
		}
	})

	t.Run("ExtensionFileAttribution", func(t *testing.T) {
		lock, err := os.ReadFile(filepath.Join(e.ws, "MODULE.bazel.lock"))
		if err != nil {
			t.Fatalf("reading generated lockfile: %v", err)
		}
		lm, err := extdeps.FromLockfile(lock)
		if err != nil {
			t.Fatal(err)
		}
		fromLock := lm.FileRepos["images.txt"]
		if len(fromLock) != 1 {
			t.Fatalf("lockfile mapping for images.txt = %v, want one repo (lockfile %s)", fromLock, lock)
		}
		snap, _ := e.load(t, "")
		repos := snap.Closures[itExtDev].Repos
		if !slices.Contains(repos, fromLock[0]) {
			t.Errorf("closure repos of %s = %v, want lockfile-derived %q", itExtDev, repos, fromLock[0])
		}
		if got := snap.Mapping.FileRepos["images.txt"]; !reflect.DeepEqual(got, fromLock) {
			t.Errorf("snapshot mapping for images.txt = %v, want %v", got, fromLock)
		}
		for _, other := range []string{itProd, itDev} {
			if r := snap.Closures[other].Repos; slices.Contains(r, fromLock[0]) {
				t.Errorf("closure repos of %s = %v, must not contain %q", other, r, fromLock[0])
			}
		}

		// Level 1: lockfile untracked, the extension-level mapping narrows it.
		prev := e.head()
		e.write("images.txt", "nginx:1.1\n")
		e.commit("bump image")
		out, errOut := e.run(0, "--since", prev)
		if want := lines(itExtDev); out != want {
			t.Errorf("level 1: got %q, want %q (stderr %s)", out, want, errOut)
		}

		// Level 2: commit the lockfile, then change images.txt together with
		// the lockfile update Bazel makes for it.
		e.git("add", "-f", "MODULE.bazel.lock")
		e.commit("track lockfile")
		prev = e.head()
		e.write("images.txt", "nginx:1.2\n")
		e.bazel("mod", "deps", "--lockfile_mode=update")
		if e.read("MODULE.bazel.lock") == string(lock) {
			t.Fatal("lockfile not updated after editing images.txt")
		}
		e.git("add", "-f", "MODULE.bazel.lock")
		sha := e.commit("bump image with lockfile")
		out, errOut = e.run(0, "--since", prev, "--format", "json")
		rep := parseJSON(t, out)
		if got, want := rep.changed(), []string{itExtDev}; !reflect.DeepEqual(got, want) {
			t.Fatalf("level 2: changed = %v, want %v (stderr %s)", got, want, errOut)
		}
		tg := rep.target(itExtDev)
		if !reflect.DeepEqual(tg.Commits, []string{sha}) {
			t.Errorf("level 2: commits = %v, want [%s]", tg.Commits, sha)
		}
		if want := []string{"MODULE.bazel.lock", "images.txt"}; !reflect.DeepEqual(tg.Files, want) {
			t.Errorf("level 2: files = %v, want %v", tg.Files, want)
		}
		if strings.Contains(errOut, "MODULE.bazel.lock is not committed") {
			t.Errorf("level 2: unexpected untracked lockfile warning: %s", errOut)
		}
	})

	t.Run("AddAndRemovePackage", func(t *testing.T) {
		prev := e.head()
		e.write("new/BUILD.bazel", "filegroup(name = \"gitops.new\", srcs = [\"n.yaml\"])\n")
		e.write("new/n.yaml", "n: 1\n")
		e.commit("add package")
		out, errOut := e.run(0, "--since", prev, "--debug-cache")
		if want := lines("//new:gitops.new"); out != want {
			t.Errorf("add: got %q, want %q (stderr %s)", out, want, errOut)
		}
		if !strings.Contains(errOut, "cache: full") {
			t.Errorf("add: want cache: full, stderr %s", errOut)
		}
		prev = e.head()
		e.git("rm", "-q", "-r", "new")
		e.commit("remove package")
		out, errOut = e.run(0, "--since", prev, "--debug-cache")
		if out != "" {
			t.Errorf("remove: got %q, want nothing (stderr %s)", out, errOut)
		}
	})
}

// run invokes the CLI on the workspace and checks its exit code.
func (e *itEnv) run(wantCode int, args ...string) (string, string) {
	e.t.Helper()
	base := []string{"--workspace", e.ws, "--cache-dir", e.cacheDir, "--targets", itUniverse}
	var out, errb bytes.Buffer
	code := run(append(base, args...), &out, &errb, func(string) string { return "" })
	if code != wantCode {
		e.t.Fatalf("run %v: exit %d, want %d\nstdout: %s\nstderr: %s", args, code, wantCode, out.String(), errb.String())
	}
	return out.String(), errb.String()
}

// load computes the graph snapshot directly; an empty cacheDir bypasses the cache.
func (e *itEnv) load(t *testing.T, cacheDir string) (*cache.Snapshot, cache.Stats) {
	t.Helper()
	snap, st, err := cache.Load(cache.Options{
		Workspace: e.ws,
		CacheDir:  cacheDir,
		Universe:  itUniverse,
		Runner:    bazelq.ExecRunner{Bazel: "bazel", Dir: e.ws},
		Repo:      gitx.Repo{Dir: e.ws},
		Warn:      func(s string) { t.Logf("cache warning: %s", s) },
	})
	if err != nil {
		t.Fatalf("cache.Load: %v", err)
	}
	return snap, st
}

func (e *itEnv) git(args ...string) string {
	e.t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)...)
	cmd.Dir = e.ws
	out, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (e *itEnv) bazel(args ...string) {
	e.t.Helper()
	cmd := exec.Command("bazel", args...)
	cmd.Dir = e.ws
	if out, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("bazel %v: %v\n%s", args, err, out)
	}
}

// commit stages every change and commits; it returns the new HEAD.
func (e *itEnv) commit(msg string) string {
	e.t.Helper()
	e.git("add", "-A")
	e.git("commit", "-q", "-m", msg)
	return e.head()
}

func (e *itEnv) head() string { e.t.Helper(); return e.git("rev-parse", "HEAD") }

func (e *itEnv) write(p, content string) {
	e.t.Helper()
	f := filepath.Join(e.ws, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func (e *itEnv) read(p string) string {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(e.ws, filepath.FromSlash(p)))
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

func (e *itEnv) remove(p string) {
	e.t.Helper()
	if err := os.Remove(filepath.Join(e.ws, filepath.FromSlash(p))); err != nil {
		e.t.Fatal(err)
	}
}

// copyTemplate copies src to dst, stripping a ".tmpl" suffix from names and
// renaming "gitignore" to ".gitignore".
func copyTemplate(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		rel = strings.TrimSuffix(rel, ".tmpl")
		if filepath.Base(rel) == "gitignore" {
			rel = filepath.Join(filepath.Dir(rel), ".gitignore")
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copying workspace template: %v", err)
	}
}

func lines(ls ...string) string {
	if len(ls) == 0 {
		return ""
	}
	return strings.Join(ls, "\n") + "\n"
}

type itReport struct {
	Targets []itTarget `json:"targets"`
	Commits map[string]struct {
		Subject string `json:"subject"`
	} `json:"commits"`
}

type itTarget struct {
	Label   string   `json:"label"`
	Status  string   `json:"status"`
	Commits []string `json:"commits"`
	Files   []string `json:"files"`
}

func parseJSON(t *testing.T, s string) itReport {
	t.Helper()
	var r itReport
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		t.Fatalf("parsing JSON output: %v\n%s", err, s)
	}
	return r
}

func (r itReport) changed() []string {
	var out []string
	for _, t := range r.Targets {
		if t.Status == "changed" {
			out = append(out, t.Label)
		}
	}
	return out
}

func (r itReport) target(label string) *itTarget {
	for i := range r.Targets {
		if r.Targets[i].Label == label {
			return &r.Targets[i]
		}
	}
	return nil
}

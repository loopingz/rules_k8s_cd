//go:build integration

// End-to-end test of the release CLI against a real Bazel, real git and a
// fake GitHub (Enterprise) API.
//
// Run with:
//
//	cd go && go test -tags integration ./release/ -run Integration -v -timeout 30m
//
// It needs `bazel` on PATH. The test workspace depends on this repository
// through a local_path_override and builds stampver with rules_go, so the
// first run downloads rules_go, the Go SDK and the other module dependencies
// (or uses a warm repository cache). The Bazel target
// //go/release:integration_test is tagged manual and is not expected to pass
// under `bazel test`: running Bazel inside the Bazel sandbox is not supported.
//
// The workspace template lives in testdata/workspace. Files there that Bazel
// would interpret in this repository (BUILD.bazel, MODULE.bazel) carry a
// ".tmpl" suffix, stripped on copy; gitignore.tmpl and bazelrc.tmpl become
// .gitignore and .bazelrc; "@@RULES_K8S_CD@@" is replaced by the absolute
// path of this repository.
//
// Components: api (//api:bin, which depends on //lib:util) and web
// (//web:site); AdoptionNoSurprise adds cli. api's release target
// //tools:record_api appends "$RELEASE_TAG <resolved //api:api.version>" to
// $IT_RELEASE_LOG, so the log lists every release target run.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"loopingz.com/rules_k8s_cd/release/pkg/release"
)

const itToken = "it-forge-token"

type itEnv struct {
	// t is the test currently running (the subtest inside e.sub); helpers
	// fail it, never a parent.
	t          *testing.T
	failed     bool   // a subtest failed; later ones are skipped
	root       string // git toplevel
	ws         string // Bazel workspace, a subdirectory of root
	origin     string // bare repository, the "origin" remote
	cacheDir   string
	releaseLog string
	forge      *fakeForge
	forgeURL   string
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in -short mode")
	}
	if _, err := exec.LookPath("bazel"); err != nil {
		t.Skip("bazel not on PATH")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	// Isolate git and Bazel from the user's configuration (~/.gitconfig,
	// ~/.bazelrc); run(), the release targets and every helper inherit this
	// env. The test's own ~/.bazelrc shares a disk cache between the
	// workspace and the release worktrees (each has its own output base, so
	// stampver and its Go toolchain would otherwise be rebuilt every time).
	// bazelisk's download cache depends on HOME: BAZELISK_HOME keeps the
	// user's.
	home := t.TempDir()
	gitconfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gitconfig, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	diskCache := filepath.Join(home, "disk_cache")
	if err := os.WriteFile(filepath.Join(home, ".bazelrc"), []byte("build --disk_cache="+diskCache+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("BAZELISK_HOME") == "" {
		if uc, err := os.UserCacheDir(); err == nil {
			t.Setenv("BAZELISK_HOME", filepath.Join(uc, "bazelisk"))
		}
	}
	logDir := t.TempDir()
	for k, v := range map[string]string{
		"HOME":                home,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_CONFIG_GLOBAL":   gitconfig,
		"GIT_AUTHOR_NAME":     "t",
		"GIT_AUTHOR_EMAIL":    "t@t",
		"GIT_COMMITTER_NAME":  "t",
		"GIT_COMMITTER_EMAIL": "t@t",
		"IT_RELEASE_LOG":      filepath.Join(logDir, "releases.log"),
	} {
		t.Setenv(k, v)
	}

	ff := newFakeForge(itToken)
	srv := httptest.NewServer(ff)
	t.Cleanup(srv.Close)

	root := t.TempDir()
	e := &itEnv{
		t:          t,
		root:       root,
		ws:         filepath.Join(root, "ws"),
		origin:     filepath.Join(t.TempDir(), "origin.git"),
		cacheDir:   t.TempDir(),
		releaseLog: os.Getenv("IT_RELEASE_LOG"),
		forge:      ff,
		forgeURL:   srv.URL,
	}
	copyTemplate(t, filepath.Join("testdata", "workspace"), e.ws, map[string]string{"@@RULES_K8S_CD@@": repoRoot})
	// Pin the workspace to this repository's Bazel version (bazelisk).
	if v, err := os.ReadFile(filepath.Join(repoRoot, ".bazelversion")); err == nil {
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
	// Release targets run in a fresh worktree (and so a fresh Bazel output
	// base) whenever HEAD is not the release commit; expunge each one once
	// its targets ran so no server or output base outlives the test.
	old := newEnv
	newEnv = func(o options, env func(string) string, out, warn func(string)) (deps, error) {
		d, err := old(o, env, out, warn)
		if err != nil || d.Env.RunTarget == nil {
			return d, err
		}
		inner := d.Env.RunTarget
		d.Env.RunTarget = func(target string, env []string, dir string) error {
			err := inner(target, env, dir)
			if dir != e.ws {
				cmd := exec.Command("bazel", "clean", "--expunge")
				cmd.Dir = dir
				if out, cerr := cmd.CombinedOutput(); cerr != nil {
					t.Logf("bazel clean --expunge in %s: %v\n%s", dir, cerr, out)
				}
			}
			return err
		}
		return d, nil
	}
	t.Cleanup(func() { newEnv = old })

	e.git("-C", e.root, "init", "-q")
	e.git("init", "-q", "--bare", e.origin)
	e.git("remote", "add", "origin", e.origin)
	if err := os.WriteFile(filepath.Join(e.root, "README.md"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.commit("chore: initial import")
	e.write("web/index.html", "<html>hello</html>\n")
	e.commit("chore(web): tweak page")

	// Subtests are order-dependent: each builds on the previous state of the
	// repository, the origin and the forge.

	e.sub(t, "FirstRelease", func(t *testing.T) {
		e.write("api/bin.sh", "#!/bin/sh\necho api v2\n")
		e.commit("feat(api): add endpoint")
		out, _ := e.run(0, "run")

		prs := e.forge.openPRs()
		if len(prs) != 1 || prs[0].Head != "release/api" {
			t.Fatalf("open PRs = %+v, want one for release/api (stdout %s)", prs, out)
		}
		pr := prs[0]
		if pr.Base != "main" || pr.Title != "chore(release): api 0.1.0" {
			t.Errorf("PR base %q title %q, want main and %q", pr.Base, pr.Title, "chore(release): api 0.1.0")
		}
		for _, s := range []string{"<!-- rules_k8s_cd:release api -->", "Initial release", "add endpoint"} {
			if !strings.Contains(pr.Body, s) {
				t.Errorf("PR body lacks %q:\n%s", s, pr.Body)
			}
		}
		e.git("fetch", "-q", "origin")
		if got := e.git("log", "-1", "--format=%s", "origin/release/api"); got != pr.Title {
			t.Errorf("release commit subject %q, want %q", got, pr.Title)
		}
		if got := e.git("rev-parse", "origin/release/api^"); got != e.head() {
			t.Errorf("release commit parent %s, want main %s", got, e.head())
		}
		if got := e.git("diff", "--name-only", "HEAD", "origin/release/api"); got != "ws/api/CHANGELOG.md\nws/api/VERSION" {
			t.Errorf("release commit changes %q, want api/CHANGELOG.md and api/VERSION", got)
		}
		if got := e.git("show", "origin/release/api:./api/VERSION"); got != "0.1.0" {
			t.Errorf("api/VERSION on the release branch = %q, want 0.1.0", got)
		}
		if tags := e.git("ls-remote", "--tags", "origin"); tags != "" {
			t.Errorf("tags on origin = %q, want none", tags)
		}
		if got := e.releases(); len(got) != 0 {
			t.Errorf("release targets ran: %v", got)
		}

		// Re-running updates the same PR.
		out, _ = e.run(0, "run")
		if prs := e.forge.openPRs(); len(prs) != 1 || prs[0].Number != pr.Number {
			t.Errorf("after rerun: open PRs = %+v, want only #%d", prs, pr.Number)
		}
		if !strings.Contains(out, fmt.Sprintf("updated PR #%d", pr.Number)) {
			t.Errorf("rerun stdout %q, want updated PR #%d", out, pr.Number)
		}
	})

	e.sub(t, "DependencyRelease", func(t *testing.T) {
		pr := e.forge.openPR("release/api")
		if pr == nil {
			t.Fatal("no open PR for api")
		}
		merged := e.squashMerge(pr)
		e.write("lib/util.sh", "util() { echo \"${1:-none}\"; }\n")
		e.commit("fix(lib): handle empty input")
		e.run(0, "run")

		if got := e.remoteTag("api@v0.1.0"); got != merged {
			t.Errorf("tag api@v0.1.0 on origin at %q, want the squash commit %s", got, merged)
		}
		rel, ok := e.forge.release("api@v0.1.0")
		if !ok {
			t.Error("no forge release api@v0.1.0")
		} else if !strings.Contains(rel.Body, "add endpoint") {
			t.Errorf("release notes lack the changelog section:\n%s", rel.Body)
		}
		if got, want := e.releases(), []string{"api@v0.1.0 0.1.0"}; !reflect.DeepEqual(got, want) {
			t.Errorf("release target runs = %q, want %q", got, want)
		}

		next := e.forge.openPR("release/api")
		if next == nil || next.Number == pr.Number || next.Title != "chore(release): api 0.1.1" {
			t.Fatalf("api PR = %+v, want a new PR for 0.1.1", next)
		}
		if !strings.Contains(next.Body, "handle empty input") || strings.Contains(next.Body, "add endpoint") {
			t.Errorf("0.1.1 PR body should list only the lib fix:\n%s", next.Body)
		}
		if web := e.forge.openPR("release/web"); web != nil {
			t.Errorf("unexpected web PR %+v", web)
		}
	})

	e.sub(t, "NoLoop", func(t *testing.T) {
		pr := e.forge.openPR("release/api")
		if pr == nil {
			t.Fatal("no open PR for api")
		}
		releaseCommit := e.mergeCommit(pr)
		out, _ := e.run(0, "run")

		if got := e.remoteTag("api@v0.1.1"); got != releaseCommit {
			t.Errorf("tag api@v0.1.1 on origin at %q, want the release commit %s", got, releaseCommit)
		}
		if _, ok := e.forge.release("api@v0.1.1"); !ok {
			t.Error("no forge release api@v0.1.1")
		}
		if got, want := e.releases(), []string{"api@v0.1.0 0.1.0", "api@v0.1.1 0.1.1"}; !reflect.DeepEqual(got, want) {
			t.Errorf("release target runs = %q, want %q", got, want)
		}
		if prs := e.forge.openPRs(); len(prs) != 0 {
			t.Errorf("open PRs after merging the release = %+v, want none (stdout %s)", prs, out)
		}
		// Release commits (squash or merged) and the VERSION/CHANGELOG they
		// change are never attributed, although api/VERSION is in api's
		// closure.
		if got := e.attributed("api"); len(got) != 0 {
			t.Errorf("commits attributed to api = %q, want none", got)
		}
		if got, want := e.attributed("web"), []string{"chore(web): tweak page", "chore: initial import"}; !reflect.DeepEqual(got, want) {
			t.Errorf("commits attributed to web = %q, want %q", got, want)
		}
	})

	e.sub(t, "IdempotentTag", func(t *testing.T) {
		calls, writes := e.forge.counts()
		e.run(0, "tag")
		if c, _ := e.forge.counts(); c != calls {
			t.Errorf("tag again made %d forge call(s), want none", c-calls)
		}
		if got := e.releases(); len(got) != 2 {
			t.Errorf("tag again ran release targets: %q", got)
		}

		e.run(0, "tag", "--retry-targets")
		if _, w := e.forge.counts(); w != writes {
			t.Errorf("tag --retry-targets made %d forge write(s), want none (the release exists)", w-writes)
		}
		if got, want := e.releases(), []string{"api@v0.1.0 0.1.0", "api@v0.1.1 0.1.1", "api@v0.1.1 0.1.1"}; !reflect.DeepEqual(got, want) {
			t.Errorf("release target runs = %q, want %q", got, want)
		}
	})

	e.sub(t, "StalePRClosed", func(t *testing.T) {
		e.write("web/extra.txt", "extra\n")
		e.commit("feat(web): add extra page")
		e.run(0, "run")
		pr := e.forge.openPR("release/web")
		if pr == nil || pr.Title != "chore(release): web 0.1.0" {
			t.Fatalf("web PR = %+v, want one for 0.1.0", pr)
		}

		build := e.read("web/BUILD.bazel")
		edited := strings.Replace(build, "    name = \"web\",\n", "    name = \"web\",\n    ignore = [\"extra.txt\"],\n", 1)
		if edited == build {
			t.Fatalf("cannot add ignore to web/BUILD.bazel:\n%s", build)
		}
		e.write("web/BUILD.bazel", edited)
		e.commit("chore(web): ignore extra.txt")

		// The feat commit only touched the now ignored file.
		if got, want := e.attributed("web"), []string{"chore(web): ignore extra.txt", "chore(web): tweak page", "chore: initial import"}; !reflect.DeepEqual(got, want) {
			t.Errorf("commits attributed to web = %q, want %q", got, want)
		}
		out, _ := e.run(0, "run")
		closed := e.forge.pr(pr.Number)
		if closed.State != "closed" {
			t.Errorf("web PR #%d state %q, want closed (stdout %s)", pr.Number, closed.State, out)
		}
		if want := []string{"No releasable changes for web anymore."}; !reflect.DeepEqual(closed.Comments, want) {
			t.Errorf("web PR comments = %q, want %q", closed.Comments, want)
		}
		if prs := e.forge.openPRs(); len(prs) != 0 {
			t.Errorf("open PRs = %+v, want none", prs)
		}
	})

	e.sub(t, "UntaggedVersion", func(t *testing.T) {
		if tags := e.git("tag", "--points-at", "HEAD"); tags != "" {
			t.Fatalf("HEAD is tagged %q; want a non-release commit", tags)
		}
		e.bazel("build", "--stamp", "//api:api.version")
		want := "main-" + e.git("rev-parse", "--short", "HEAD") + "\n"
		if got := e.read("bazel-bin/api/api.version.txt"); got != want {
			t.Errorf("//api:api.version = %q, want %q", got, want)
		}
	})

	e.sub(t, "AdoptionNoSurprise", func(t *testing.T) {
		// A component adopted with its existing tag: VERSION matches the tag,
		// but there is no forge release and nothing changed since.
		e.write("cli/BUILD.bazel", `load("@rules_k8s_cd//lib:release.bzl", "release_component")

filegroup(
    name = "files",
    srcs = ["main.txt"],
)

release_component(
    name = "cli",
    release_targets = ["//tools:record_cli"],
    targets = [":files"],
    visibility = ["//visibility:public"],
)
`)
		e.write("cli/main.txt", "cli\n")
		e.write("cli/VERSION", "1.0.0\n")
		e.write("tools/BUILD.bazel", e.read("tools/BUILD.bazel")+`
sh_binary(
    name = "record_cli",
    srcs = ["record.sh"],
    args = ["cli/cli.version.txt"],
    data = ["//cli:cli.version"],
)
`)
		sha := e.commit("chore: adopt the cli component")
		e.git("tag", "-a", "cli@v1.0.0", sha, "-m", "cli 1.0.0")
		e.git("push", "-q", "origin", "refs/tags/cli@v1.0.0")

		_, writes := e.forge.counts()
		before := e.releases()
		out, _ := e.run(0, "run")
		if _, w := e.forge.counts(); w != writes {
			t.Errorf("run made %d forge write(s), want none (stdout %s)", w-writes, out)
		}
		if _, ok := e.forge.release("cli@v1.0.0"); ok {
			t.Error("a forge release was created for the pre-existing tag cli@v1.0.0")
		}
		if got := e.releases(); !reflect.DeepEqual(got, before) {
			t.Errorf("release targets ran: %q, want %q", got, before)
		}
		if got := e.git("ls-remote", "origin", "refs/heads/release/cli"); got != "" {
			t.Errorf("release branch pushed for cli: %q", got)
		}
		if strings.Contains(out, "tagged") || strings.Contains(out, "opened") {
			t.Errorf("unexpected action in stdout:\n%s", out)
		}
	})
}

// sub runs fn as a subtest with e.t set to it. Subtests share one workspace
// and are order-dependent, so after a failure the remaining ones are skipped.
func (e *itEnv) sub(t *testing.T, name string, fn func(t *testing.T)) {
	t.Run(name, func(t *testing.T) {
		if e.failed {
			t.Skip("earlier subtest failed; shared workspace state is unreliable")
		}
		parent := e.t
		e.t = t
		defer func() {
			e.t = parent
			if t.Failed() {
				e.failed = true
			}
		}()
		fn(t)
	})
}

// env is the CLI's environment: only the forge token.
func (e *itEnv) env(k string) string {
	if k == "FORGE_TOKEN" {
		return itToken
	}
	return ""
}

// run invokes the CLI command (args[0]) on the workspace, against the fake
// forge, and checks its exit code.
func (e *itEnv) run(wantCode int, args ...string) (string, string) {
	e.t.Helper()
	full := append([]string{args[0],
		"--workspace", e.ws, "--cache-dir", e.cacheDir,
		"--forge", "github", "--forge-url", e.forgeURL, "--repo", "o/r",
	}, args[1:]...)
	var out, errb bytes.Buffer
	code := run(full, &out, &errb, e.env)
	e.t.Logf("release %s:\nstdout:\n%sstderr:\n%s", strings.Join(args, " "), out.String(), errb.String())
	if code != wantCode {
		e.t.Fatalf("release %v: exit %d, want %d\nstdout: %s\nstderr: %s", args, code, wantCode, out.String(), errb.String())
	}
	if strings.Contains(out.String()+errb.String(), itToken) {
		e.t.Errorf("release %v printed the forge token", args)
	}
	if bad := e.forge.problems(); len(bad) > 0 {
		e.t.Errorf("fake forge: %s", strings.Join(bad, "; "))
	}
	return out.String(), errb.String()
}

// attributed returns the subjects of the commits release status attributes
// to component name, newest first.
func (e *itEnv) attributed(name string) []string {
	e.t.Helper()
	o := options{command: "status", workspace: e.ws, cacheDir: e.cacheDir, bazel: "bazel", remote: "origin"}
	d, err := defaultNewEnv(o, e.env, func(string) {}, func(s string) { e.t.Logf("warning: %s", s) })
	if err != nil {
		e.t.Fatal(err)
	}
	comps, err := d.Discover("//...")
	if err != nil {
		e.t.Fatal(err)
	}
	plans, err := release.Status(d.Env, comps)
	if err != nil {
		e.t.Fatal(err)
	}
	for _, p := range plans {
		if p.Component.Name != name {
			continue
		}
		out := []string{}
		for _, en := range p.Entries {
			out = append(out, en.Commit.Subject)
		}
		return out
	}
	e.t.Fatalf("no plan for %s", name)
	return nil
}

// squashMerge merges pr like a forge squash merge: one commit on main with
// the release branch's tree and the subject "<title> (#N)". It returns the
// new main commit.
func (e *itEnv) squashMerge(pr *fakePR) string {
	e.t.Helper()
	e.git("fetch", "-q", "origin")
	branch := "origin/" + pr.Head
	if parent := e.git("rev-parse", branch+"^"); parent != e.head() {
		e.t.Fatalf("%s is based on %s, not main %s", branch, parent, e.head())
	}
	sha := e.git("commit-tree", branch+"^{tree}", "-p", "HEAD", "-m", fmt.Sprintf("%s (#%d)", pr.Title, pr.Number))
	e.git("merge", "-q", "--ff-only", sha)
	e.git("push", "-q", "origin", "main")
	e.forge.markMerged(pr.Number)
	return sha
}

// mergeCommit merges pr with a merge commit (--no-ff) and returns the
// release branch commit.
func (e *itEnv) mergeCommit(pr *fakePR) string {
	e.t.Helper()
	e.git("fetch", "-q", "origin")
	branch := "origin/" + pr.Head
	sha := e.git("rev-parse", branch)
	e.git("merge", "-q", "--no-ff", "-m", fmt.Sprintf("Merge pull request #%d from o/%s", pr.Number, pr.Head), branch)
	e.git("push", "-q", "origin", "main")
	e.forge.markMerged(pr.Number)
	return sha
}

// remoteTag returns the commit tag points to on origin, "" if absent.
func (e *itEnv) remoteTag(tag string) string {
	e.t.Helper()
	out := e.git("ls-remote", "origin", "refs/tags/"+tag+"^{}")
	if f := strings.Fields(out); len(f) == 2 {
		return f[0]
	}
	return ""
}

// releases returns the lines of the release target log.
func (e *itEnv) releases() []string {
	e.t.Helper()
	b, err := os.ReadFile(e.releaseLog)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		e.t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func (e *itEnv) git(args ...string) string {
	e.t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)...)
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

// commit stages every change, commits and pushes main; it returns the new
// HEAD.
func (e *itEnv) commit(msg string) string {
	e.t.Helper()
	e.git("add", "-A")
	e.git("commit", "-q", "-m", msg)
	e.git("push", "-q", "origin", "HEAD:refs/heads/main")
	e.git("fetch", "-q", "origin")
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

// copyTemplate copies src to dst, stripping a ".tmpl" suffix from names,
// renaming "gitignore" and "bazelrc" to their dotfiles, applying subs to
// every file and keeping the executable bit.
func copyTemplate(t *testing.T, src, dst string, subs map[string]string) {
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
		if b := filepath.Base(rel); b == "gitignore" || b == "bazelrc" {
			rel = filepath.Join(filepath.Dir(rel), "."+b)
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for k, v := range subs {
			data = bytes.ReplaceAll(data, []byte(k), []byte(v))
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644|(info.Mode().Perm()&0o111))
	})
	if err != nil {
		t.Fatalf("copying workspace template: %v", err)
	}
}

// fakeForge is an in-memory GitHub Enterprise API (served under /api/v3)
// for repository o/r, implementing the endpoints of forge.NewGitHub.
type fakeForge struct {
	token string

	mu       sync.Mutex
	next     int
	prs      map[int]*fakePR
	releases map[string]fakeRelease
	calls    int // every request
	writes   int // every non-GET request
	bad      []string
}

type fakePR struct {
	Number                  int
	Head, Base, Title, Body string
	State                   string // open, closed or merged
	Comments                []string
}

type fakeRelease struct{ Name, Body string }

func newFakeForge(token string) *fakeForge {
	return &fakeForge{token: token, next: 1, prs: map[int]*fakePR{}, releases: map[string]fakeRelease{}}
}

func (f *fakeForge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if r.Method != http.MethodGet {
		f.writes++
	}
	if got := r.Header.Get("Authorization"); got != "Bearer "+f.token {
		f.bad = append(f.bad, fmt.Sprintf("%s %s: Authorization %q", r.Method, r.URL.Path, got))
		http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
		return
	}
	const prefix = "/api/v3/repos/o/r/"
	path := r.URL.EscapedPath()
	if !strings.HasPrefix(path, prefix) {
		f.notFound(w, r)
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	var in map[string]any
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		if len(b) > 0 {
			if err := json.Unmarshal(b, &in); err != nil {
				f.bad = append(f.bad, fmt.Sprintf("%s %s: invalid JSON body", r.Method, path))
			}
		}
	}
	str := func(k string) string { s, _ := in[k].(string); return s }

	switch {
	case len(parts) == 1 && parts[0] == "pulls" && r.Method == http.MethodGet:
		q := r.URL.Query()
		head := strings.TrimPrefix(q.Get("head"), "o:")
		out := []any{}
		for _, n := range f.sortedPRs() {
			p := f.prs[n]
			if q.Get("state") == "open" && p.State != "open" {
				continue
			}
			if head != "" && p.Head != head {
				continue
			}
			out = append(out, p.json())
		}
		writeJSON(w, http.StatusOK, out)
	case len(parts) == 1 && parts[0] == "pulls" && r.Method == http.MethodPost:
		for _, p := range f.prs {
			if p.State == "open" && p.Head == str("head") {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": "A pull request already exists"})
				return
			}
		}
		p := &fakePR{Number: f.next, Head: str("head"), Base: str("base"), Title: str("title"), Body: str("body"), State: "open"}
		f.next++
		f.prs[p.Number] = p
		writeJSON(w, http.StatusCreated, p.json())
	case len(parts) == 2 && parts[0] == "pulls" && r.Method == http.MethodPatch:
		p := f.prByPath(parts[1])
		if p == nil {
			f.notFound(w, r)
			return
		}
		if v, ok := in["title"].(string); ok {
			p.Title = v
		}
		if v, ok := in["body"].(string); ok {
			p.Body = v
		}
		if str("state") == "closed" && p.State == "open" {
			p.State = "closed"
		}
		writeJSON(w, http.StatusOK, p.json())
	case len(parts) == 3 && parts[0] == "issues" && parts[2] == "comments" && r.Method == http.MethodPost:
		p := f.prByPath(parts[1])
		if p == nil {
			f.notFound(w, r)
			return
		}
		p.Comments = append(p.Comments, str("body"))
		writeJSON(w, http.StatusCreated, map[string]any{"body": str("body")})
	case len(parts) == 3 && parts[0] == "releases" && parts[1] == "tags" && r.Method == http.MethodGet:
		tag, err := url.PathUnescape(parts[2])
		if err != nil {
			f.notFound(w, r)
			return
		}
		rel, ok := f.releases[tag]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not Found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tag_name": tag, "name": rel.Name, "body": rel.Body})
	case len(parts) == 1 && parts[0] == "releases" && r.Method == http.MethodPost:
		tag := str("tag_name")
		if _, ok := f.releases[tag]; ok {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": "Validation Failed", "errors": []any{map[string]any{"code": "already_exists"}}})
			return
		}
		f.releases[tag] = fakeRelease{Name: str("name"), Body: str("body")}
		writeJSON(w, http.StatusCreated, map[string]any{"tag_name": tag})
	default:
		f.notFound(w, r)
	}
}

func (f *fakeForge) notFound(w http.ResponseWriter, r *http.Request) {
	f.bad = append(f.bad, fmt.Sprintf("unexpected request %s %s", r.Method, r.URL))
	writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not Found"})
}

func (f *fakeForge) prByPath(s string) *fakePR {
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return f.prs[n]
}

func (f *fakeForge) sortedPRs() []int {
	ns := make([]int, 0, len(f.prs))
	for n := range f.prs {
		ns = append(ns, n)
	}
	sort.Ints(ns)
	return ns
}

func (p *fakePR) json() map[string]any {
	return map[string]any{
		"number":   p.Number,
		"title":    p.Title,
		"body":     p.Body,
		"state":    map[bool]string{true: "open", false: "closed"}[p.State == "open"],
		"html_url": fmt.Sprintf("https://ghe.example/o/r/pull/%d", p.Number),
		"head":     map[string]any{"ref": p.Head},
		"base":     map[string]any{"ref": p.Base},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// openPRs returns copies of the open PRs, by number.
func (f *fakeForge) openPRs() []fakePR {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []fakePR
	for _, n := range f.sortedPRs() {
		if p := f.prs[n]; p.State == "open" {
			out = append(out, *p)
		}
	}
	return out
}

// openPR returns a copy of the open PR of head, or nil.
func (f *fakeForge) openPR(head string) *fakePR {
	for _, p := range f.openPRs() {
		if p.Head == head {
			return &p
		}
	}
	return nil
}

// pr returns a copy of PR n (zero if absent).
func (f *fakeForge) pr(n int) fakePR {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p := f.prs[n]; p != nil {
		return *p
	}
	return fakePR{}
}

func (f *fakeForge) markMerged(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prs[n].State = "merged"
}

func (f *fakeForge) release(tag string) (fakeRelease, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.releases[tag]
	return r, ok
}

func (f *fakeForge) counts() (calls, writes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, f.writes
}

// problems returns and clears the unexpected requests seen.
func (f *fakeForge) problems() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	bad := f.bad
	f.bad = nil
	return bad
}

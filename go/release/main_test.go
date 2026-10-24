package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/release/pkg/attrib"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
	"loopingz.com/rules_k8s_cd/release/pkg/release"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

const secret = "s3cr3t-t0ken-value"

func runCLI(env map[string]string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb, func(k string) string { return env[k] })
	return code, out.String(), errb.String()
}

// withNewEnv replaces the newEnv hook for one test.
func withNewEnv(t *testing.T, f func(o options, env func(string) string, out, warn func(string)) (deps, error)) {
	t.Helper()
	old := newEnv
	newEnv = f
	t.Cleanup(func() { newEnv = old })
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, b)
	}
	return strings.TrimSpace(string(b))
}

func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
}

func write(t *testing.T, dir, path, content string) {
	t.Helper()
	p := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"bogus"},
		{"status", "--nope"},
		{"status", "extra"},
		{"--dry-run", "status"},
	} {
		if code, _, _ := runCLI(nil, args...); code != 2 {
			t.Errorf("run(%v) = %d, want 2", args, code)
		}
	}
	if code, out, _ := runCLI(nil, "--help"); code != 0 || !strings.Contains(out, "usage:") {
		t.Errorf("--help: %d %q", code, out)
	}
	if code, _, _ := runCLI(nil, "status", "-h"); code != 0 {
		t.Errorf("status -h = %d, want 0", code)
	}
}

// statusFixture builds a repository with four components in the states
// release PR (a), up to date (b), first release (c) and awaiting tag (d).
func statusFixture(t *testing.T) (release.Env, []component.Component) {
	t.Helper()
	isolateGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "a/VERSION", "1.0.0\n")
	write(t, dir, "b/VERSION", "1.0.0\n")
	write(t, dir, "d/VERSION", "0.1.0\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	git(t, dir, "tag", "a@v1.0.0")
	git(t, dir, "tag", "b@v1.0.0")
	write(t, dir, "d/VERSION", "1.1.0\n")
	git(t, dir, "commit", "-qam", "chore(release): d 1.1.0")

	mk := func(n string) component.Component {
		c := component.Component{Label: "//" + n + ":rel", Name: n, Package: n, Initial: semver.Version{Minor: 1}, TagFormat: "{name}@v{version}"}
		c.VersionFile = n + "/VERSION"
		return c
	}
	c := mk("c")
	comps := []component.Component{mk("d"), c, mk("b"), mk("a")}
	feat := gitx.Commit{SHA: "1111111", Short: "1111111", Subject: "feat: new thing"}
	e := release.Env{
		Repo: gitx.Repo{Dir: dir},
		Attribute: func(o cache.Options, repo gitx.Repo, head string, cs []component.Component, st map[string]component.State, g []string) (map[string]attrib.Result, []string, error) {
			res := map[string]attrib.Result{}
			for _, c := range cs {
				r := attrib.Result{Component: c, Files: map[string][]string{}}
				if c.Name == "a" || c.Name == "c" {
					r.Commits = []gitx.Commit{feat}
					r.Files[feat.SHA] = []string{c.Package + "/x.go"}
				}
				res[c.Label] = r
			}
			return res, nil, nil
		},
	}
	return e, comps
}

func TestStatusTable(t *testing.T) {
	e, comps := statusFixture(t)
	withNewEnv(t, func(o options, env func(string) string, out, warn func(string)) (deps, error) {
		if o.needsForge {
			t.Error("status must not need a forge")
		}
		e.Out, e.Warn = out, warn
		return deps{Env: e, Discover: func(scope string) ([]component.Component, error) {
			if scope != "//..." {
				t.Errorf("scope = %q", scope)
			}
			return comps, nil
		}}, nil
	})
	code, out, errb := runCLI(nil, "status")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, errb)
	}
	want := `component | current | next  | state
a         | 1.0.0   | 1.1.0 | release PR
b         | 1.0.0   | -     | up to date
c         | -       | 0.1.0 | first release
d         | 1.1.0   | -     | awaiting tag
`
	if out != want {
		t.Errorf("table:\n%s\nwant:\n%s", out, want)
	}
}

func TestRunSkipsPRWhenTagFails(t *testing.T) {
	e, comps := statusFixture(t)
	e.Remote = "" // Tag requires a remote and fails
	withNewEnv(t, func(o options, env func(string) string, out, warn func(string)) (deps, error) {
		e.Out, e.Warn = out, warn
		return deps{Env: e, Discover: func(string) ([]component.Component, error) { return comps, nil }}, nil
	})
	code, _, errb := runCLI(nil, "run")
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(errb, "release tag") || !strings.Contains(errb, "skipping pr") || strings.Contains(errb, "release pr") {
		t.Errorf("stderr: %s", errb)
	}
}

func TestRunDoesPRAfterTag(t *testing.T) {
	e, comps := statusFixture(t)
	e.Remote, e.MainBranch, e.DryRun = "origin", "main", true
	origin := t.TempDir()
	git(t, origin, "init", "-q", "--bare")
	git(t, e.Repo.Dir, "remote", "add", "origin", origin)
	git(t, e.Repo.Dir, "push", "-q", "origin", "HEAD:refs/heads/main", "--tags")
	git(t, e.Repo.Dir, "fetch", "-q", "origin")
	withNewEnv(t, func(o options, env func(string) string, out, warn func(string)) (deps, error) {
		e.Out, e.Warn = out, warn
		return deps{Env: e, Discover: func(string) ([]component.Component, error) { return comps, nil }}, nil
	})
	// Dry-run without forge: tag has nothing published to find, pr prints.
	code, out, errb := runCLI(nil, "run", "--dry-run")
	if code != 0 {
		t.Fatalf("code %d stderr %s", code, errb)
	}
	tagAt, prAt := strings.Index(out, "d: would tag d@v1.1.0"), strings.Index(out, "a: would push release/a")
	if tagAt < 0 || prAt < 0 || tagAt > prAt {
		t.Errorf("want tag output before pr output:\n%s", out)
	}
}

func TestFlagsReachOptions(t *testing.T) {
	var got options
	withNewEnv(t, func(o options, env func(string) string, out, warn func(string)) (deps, error) {
		got = o
		return deps{}, errors.New("stop")
	})
	code, _, _ := runCLI(nil, "tag", "--forge", "forgejo", "--forge-url", "https://f", "--repo", "o/r",
		"--remote", "up", "--main-branch", "dev", "--components", "//x/...", "--dry-run", "--retry-targets",
		"--cache-dir", "c", "--workspace", "w", "--bazel", "bz", "--global-file", "g1", "--global-file", "g2", "--ignore", "i")
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	want := options{command: "tag", forge: "forgejo", forgeURL: "https://f", repo: "o/r", remote: "up", mainBranch: "dev",
		components: "//x/...", dryRun: true, retryTargets: true, cacheDir: "c", workspace: "w", bazel: "bz",
		globalFiles: stringSlice{"g1", "g2"}, ignores: stringSlice{"i"}, needsForge: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("options:\n%+v\nwant\n%+v", got, want)
	}
	runCLI(nil, "status")
	if got.remote != "origin" || got.components != "//..." || got.needsForge {
		t.Errorf("defaults: %+v", got)
	}
}

func TestTokenNeverPrinted(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "f", "x")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "init")
	tok := map[string]string{"FORGE_TOKEN": secret, "GITHUB_TOKEN": secret + "-gh", "FORGEJO_TOKEN": secret + "-fj"}
	for _, args := range [][]string{
		{"pr", "--workspace", dir, "--forge", "bogus"},
		{"pr", "--workspace", dir, "--forge", "github"},                                                    // no owner/repo
		{"tag", "--workspace", dir, "--forge", "forgejo", "--repo", "o/r"},                                 // no URL
		{"pr", "--workspace", dir, "--forge", "github", "--repo", "o/r", "--bazel", "/nonexistent/bazel"},  // discovery fails
		{"run", "--workspace", dir, "--forge", "github", "--repo", "o/r", "--bazel", "/nonexistent/bazel"}, // discovery fails
		{"status", "--workspace", dir + "/missing"},
		{"pr", "--workspace", dir, "--forge", "github", "--forge-url", "https://" + secret + "@h", "--repo", "o/r", "--bazel", "/nonexistent/bazel"},
	} {
		code, out, errb := runCLI(tok, args...)
		if code != 1 {
			t.Errorf("%v: code %d, want 1 (stderr %s)", args, code, errb)
		}
		if strings.Contains(out+errb, secret) {
			t.Errorf("%v: token leaked:\nstdout: %s\nstderr: %s", args, out, errb)
		}
	}
	// Errors that embed a token are masked.
	withNewEnv(t, func(o options, env func(string) string, out, warn func(string)) (deps, error) {
		warn("saw " + secret)
		return deps{}, errors.New("boom " + secret)
	})
	code, out, errb := runCLI(tok, "pr")
	if code != 1 || strings.Contains(out+errb, secret) || !strings.Contains(errb, "boom ***") {
		t.Errorf("scrub: code %d out %q err %q", code, out, errb)
	}
}

func TestMissingToken(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "f", "x")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "init")
	base := []string{"--workspace", dir, "--forge", "github", "--repo", "o/r", "--bazel", "/nonexistent/bazel"}
	for _, cmd := range []string{"pr", "tag", "run"} {
		code, _, errb := runCLI(nil, append([]string{cmd}, base...)...)
		if code != 1 || !strings.Contains(errb, "no forge token") {
			t.Errorf("%s: code %d stderr %s", cmd, code, errb)
		}
	}
	// Dry-run proceeds (it only fails later, at the missing bazel).
	code, _, errb := runCLI(nil, append([]string{"pr", "--dry-run"}, base...)...)
	if strings.Contains(errb, "no forge token:") || !strings.Contains(errb, "dry-run continues without forge lookups") {
		t.Errorf("dry-run: code %d stderr %s", code, errb)
	}
	// Status never needs a token.
	_, _, errb = runCLI(nil, append([]string{"status"}, base...)...)
	if strings.Contains(errb, "token") {
		t.Errorf("status asked for a token: %s", errb)
	}
}

func TestDefaultMainBranch(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "f", "x")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "init")
	repo := gitx.Repo{Dir: dir}
	if got := defaultMainBranch(repo, "origin"); got != "main" {
		t.Errorf("no remote HEAD: %q", got)
	}
	git(t, dir, "update-ref", "refs/remotes/origin/trunk", "HEAD")
	git(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	if got := defaultMainBranch(repo, "origin"); got != "trunk" {
		t.Errorf("origin/HEAD -> trunk: %q", got)
	}
	if got := defaultMainBranch(repo, "other"); got != "main" {
		t.Errorf("other remote: %q", got)
	}
}

func TestIgnoringDropsFullyIgnoredCommits(t *testing.T) {
	c1 := gitx.Commit{SHA: "1", Subject: "feat: a"}
	c2 := gitx.Commit{SHA: "2", Subject: "fix: b"}
	c3 := gitx.Commit{SHA: "3", Subject: "fix: c"}
	inner := func(cache.Options, gitx.Repo, string, []component.Component, map[string]component.State, []string) (map[string]attrib.Result, []string, error) {
		return map[string]attrib.Result{"//a:r": {Commits: []gitx.Commit{c1, c2, c3}, Files: map[string][]string{
			"1": {"docs/x.md"},
			"2": {"docs/y.md", "src/y.go"},
			"3": nil,
		}}}, nil, nil
	}
	res, _, err := ignoring(inner, []string{"docs/**"})(cache.Options{}, gitx.Repo{}, "HEAD", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := res["//a:r"]
	if len(r.Commits) != 2 || r.Commits[0].SHA != "2" || r.Commits[1].SHA != "3" {
		t.Errorf("commits: %+v", r.Commits)
	}
	if !reflect.DeepEqual(r.Files["2"], []string{"src/y.go"}) {
		t.Errorf("files: %v", r.Files)
	}
}

package gitwrite

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-c", "user.name=T", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "HOME="+dir,
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
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

// setup returns a repo (toplevel) with an initial commit and a bare origin.
func setup(t *testing.T) (top, origin string) {
	t.Helper()
	// Isolate git config for the code under test too.
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	top = t.TempDir()
	origin = t.TempDir()
	git(t, origin, "init", "-q", "--bare")
	git(t, top, "init", "-q")
	git(t, top, "remote", "add", "origin", origin)
	write(t, top, "ws/keep.txt", "keep\n")
	write(t, top, "ws/VERSION", "1.0.0\n")
	write(t, top, "other/o.txt", "o\n")
	git(t, top, "add", "-A")
	git(t, top, "commit", "-q", "-m", "init")
	return top, origin
}

func testCommitFiles(t *testing.T, sub string) {
	top, _ := setup(t)
	dir := filepath.Join(top, sub)
	repo := gitx.Repo{Dir: dir}
	prefix := ""
	if sub != "" {
		prefix = sub + "/"
	}
	parent := git(t, top, "rev-parse", "HEAD")
	head := parent
	write(t, top, "untracked.txt", "u")
	statusBefore := git(t, top, "status", "--porcelain")

	sha, err := CommitFiles(repo, parent, map[string][]byte{
		"VERSION":       []byte("1.1.0\n"),
		"new/CHANGELOG": []byte("# c\n"),
	}, "chore(release): x 1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if got := git(t, top, "rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD moved")
	}
	if got := git(t, top, "status", "--porcelain"); got != statusBefore {
		t.Errorf("status changed: %q -> %q", statusBefore, got)
	}
	if got := git(t, top, "rev-parse", sha+"^"); got != parent {
		t.Errorf("parent = %s", got)
	}
	if got := git(t, top, "log", "-1", "--format=%s", sha); got != "chore(release): x 1.1.0" {
		t.Errorf("subject %q", got)
	}
	if got := git(t, top, "show", sha+":"+prefix+"VERSION"); got != "1.1.0" {
		t.Errorf("VERSION %q", got)
	}
	if got := git(t, top, "show", sha+":"+prefix+"new/CHANGELOG"); got != "# c" {
		t.Errorf("CHANGELOG %q", got)
	}
	if got := git(t, top, "show", sha+":other/o.txt"); got != "o" {
		t.Errorf("other changed: %q", got)
	}
	if sub == "ws" {
		if got := git(t, top, "show", sha+":ws/keep.txt"); got != "keep" {
			t.Errorf("keep: %q", got)
		}
	}
	if _, err := os.Stat(filepath.Join(top, "untracked.txt")); err != nil {
		t.Error(err)
	}
}

func TestCommitFilesToplevel(t *testing.T) { testCommitFiles(t, "") }
func TestCommitFilesSubdir(t *testing.T)   { testCommitFiles(t, "ws") }

func TestCommitFilesRejectsEscape(t *testing.T) {
	top, _ := setup(t)
	parent := git(t, top, "rev-parse", "HEAD")
	for _, p := range []string{"../x", "/abs", ""} {
		if _, err := CommitFiles(gitx.Repo{Dir: top}, parent, map[string][]byte{p: []byte("x")}, "m"); err == nil {
			t.Errorf("path %q accepted", p)
		}
	}
}

func TestCommitFilesIdentityFallback(t *testing.T) {
	top, _ := setup(t)
	parent := git(t, top, "rev-parse", "HEAD")
	git(t, top, "config", "user.useConfigOnly", "true")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	t.Setenv("HOME", t.TempDir())
	if _, err := exec.Command("git", "-C", top, "var", "GIT_COMMITTER_IDENT").Output(); err == nil {
		t.Fatal("test setup: git still resolves an identity")
	}
	sha, err := CommitFiles(gitx.Repo{Dir: top}, parent, map[string][]byte{"a": []byte("a")}, "m")
	if err != nil {
		t.Fatal(err)
	}
	if got := git(t, top, "log", "-1", "--format=%an <%ae>", sha); got != "rules_k8s_cd release <release@localhost>" {
		t.Errorf("identity %q", got)
	}
}

func TestCommitFilesRejectsDuplicates(t *testing.T) {
	top, _ := setup(t)
	parent := git(t, top, "rev-parse", "HEAD")
	_, err := CommitFiles(gitx.Repo{Dir: top}, parent, map[string][]byte{"a/b": nil, "a/./b": nil}, "m")
	if err == nil {
		t.Error("duplicate cleaned paths accepted")
	}
}

func TestHostileConfig(t *testing.T) {
	top, origin := setup(t)
	git(t, top, "config", "tag.gpgSign", "true")
	git(t, top, "config", "commit.gpgsign", "true")
	git(t, top, "config", "gpg.program", "/bin/false")
	hook := filepath.Join(top, ".git", "hooks", "pre-push")
	write(t, top, ".git/hooks/pre-push", "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
	repo := gitx.Repo{Dir: top}
	parent := git(t, top, "rev-parse", "HEAD")
	sha, err := CommitFiles(repo, parent, map[string][]byte{"f": []byte("a")}, "m")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := PushBranch(repo, "origin", sha, "release/x"); err != nil {
		t.Fatalf("push: %v", err)
	}
	if err := CreateTag(repo, "origin", "x@v1", sha, "m"); err != nil {
		t.Fatalf("tag: %v", err)
	}
	if got := git(t, origin, "rev-parse", "refs/heads/release/x"); got != sha {
		t.Errorf("branch = %s", got)
	}
}

func TestPushBranchValidation(t *testing.T) {
	top, origin := setup(t)
	repo := gitx.Repo{Dir: top}
	sha := git(t, top, "rev-parse", "HEAD")
	for _, b := range []string{"main", "a:b", "release/a b", "release/../x", "-x", "+release/x", "release/"} {
		if err := PushBranch(repo, "origin", sha, b); err == nil {
			t.Errorf("branch %q accepted", b)
		}
	}
	if err := PushBranch(repo, "-x", sha, "release/web"); err == nil {
		t.Error("remote -x accepted")
	}
	if err := PushBranch(repo, "origin", "-x", "release/web"); err == nil {
		t.Error("sha -x accepted")
	}
	if err := PushBranch(repo, "origin", sha, "release/web"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, origin, "rev-parse", "refs/heads/release/web"); got != sha {
		t.Errorf("origin = %s", got)
	}
}

func TestCreateTagValidation(t *testing.T) {
	top, _ := setup(t)
	repo := gitx.Repo{Dir: top}
	sha := git(t, top, "rev-parse", "HEAD")
	for _, tag := range []string{"-x", "a b", "a..b", "x:y", ""} {
		if err := CreateTag(repo, "origin", tag, sha, "m"); err == nil {
			t.Errorf("tag %q accepted", tag)
		}
	}
	if err := CreateTag(repo, "-o", "t", sha, "m"); err == nil {
		t.Error("remote -o accepted")
	}
}

func TestCreateTagFreshClone(t *testing.T) {
	top, origin := setup(t)
	repo := gitx.Repo{Dir: top}
	c1 := git(t, top, "rev-parse", "HEAD")
	if err := CreateTag(repo, "origin", "x@v1", c1, "m"); err != nil {
		t.Fatal(err)
	}
	clone := t.TempDir()
	git(t, clone, "clone", "-q", "--no-tags", origin, ".")
	crepo := gitx.Repo{Dir: clone}
	if _, ok, err := crepo.TagCommit("x@v1"); err != nil || ok {
		t.Fatalf("tag present locally before CreateTag: ok=%v err=%v", ok, err)
	}
	if err := CreateTag(crepo, "origin", "x@v1", c1, "m"); err != nil {
		t.Fatalf("fresh clone same commit: %v", err)
	}
	if _, ok, _ := crepo.TagCommit("x@v1"); ok {
		t.Error("CreateTag created a local tag although the remote already had it")
	}
	c2, err := CommitFiles(crepo, c1, map[string][]byte{"f": []byte("a")}, "a")
	if err != nil {
		t.Fatal(err)
	}
	err = CreateTag(crepo, "origin", "x@v1", c2, "m")
	if err == nil || !strings.Contains(err.Error(), c1) || !strings.Contains(err.Error(), c2) {
		t.Errorf("remote conflict error = %v", err)
	}
}

func TestPushBranchForce(t *testing.T) {
	top, origin := setup(t)
	repo := gitx.Repo{Dir: top}
	parent := git(t, top, "rev-parse", "HEAD")
	a, err := CommitFiles(repo, parent, map[string][]byte{"f": []byte("a")}, "a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := CommitFiles(repo, parent, map[string][]byte{"f": []byte("b")}, "b")
	if err != nil {
		t.Fatal(err)
	}
	if err := PushBranch(repo, "origin", a, "release/x"); err != nil {
		t.Fatal(err)
	}
	if err := PushBranch(repo, "origin", b, "release/x"); err != nil { // non-fast-forward
		t.Fatal(err)
	}
	if got := git(t, origin, "rev-parse", "refs/heads/release/x"); got != b {
		t.Errorf("origin = %s want %s", got, b)
	}
}

func TestCreateTag(t *testing.T) {
	top, origin := setup(t)
	repo := gitx.Repo{Dir: top}
	c1 := git(t, top, "rev-parse", "HEAD")
	c2, err := CommitFiles(repo, c1, map[string][]byte{"f": []byte("a")}, "a")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateTag(repo, "origin", "x@v1.0.0", c1, "rel"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, origin, "rev-parse", "x@v1.0.0^{commit}"); got != c1 {
		t.Errorf("origin tag = %s", got)
	}
	if got := git(t, top, "cat-file", "-t", "refs/tags/x@v1.0.0"); got != "tag" {
		t.Errorf("not annotated: %s", got)
	}
	// Idempotent, also repairs a missing remote tag.
	git(t, origin, "tag", "-d", "x@v1.0.0")
	if err := CreateTag(repo, "origin", "x@v1.0.0", c1, "rel"); err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if got := git(t, origin, "rev-parse", "x@v1.0.0^{commit}"); got != c1 {
		t.Errorf("origin tag after rerun = %s", got)
	}
	err = CreateTag(repo, "origin", "x@v1.0.0", c2, "rel")
	if err == nil || !strings.Contains(err.Error(), c1) || !strings.Contains(err.Error(), c2) {
		t.Errorf("conflict error = %v", err)
	}
}

func TestLastCommitTouching(t *testing.T) {
	top, _ := setup(t)
	write(t, top, "ws/keep.txt", "2\n")
	git(t, top, "add", "-A")
	git(t, top, "commit", "-q", "-m", "touch keep")
	want := git(t, top, "rev-parse", "HEAD")
	write(t, top, "other/o.txt", "2\n")
	git(t, top, "add", "-A")
	git(t, top, "commit", "-q", "-m", "touch other")

	repo := gitx.Repo{Dir: filepath.Join(top, "ws")}
	c, ok, err := LastCommitTouching(repo, "keep.txt")
	if err != nil || !ok || c.SHA != want || c.Subject != "touch keep" {
		t.Errorf("got %+v %v %v", c, ok, err)
	}
	if _, ok, err := LastCommitTouching(repo, "missing"); err != nil || ok {
		t.Errorf("missing: ok=%v err=%v", ok, err)
	}
}

func TestCreateTagBogusRemote(t *testing.T) {
	top, _ := setup(t)
	sha := git(t, top, "rev-parse", "HEAD")
	if err := CreateTag(gitx.Repo{Dir: top}, filepath.Join(top, "nonexistent"), "t", sha, "m"); err == nil {
		t.Error("expected error for unreachable remote")
	}
}

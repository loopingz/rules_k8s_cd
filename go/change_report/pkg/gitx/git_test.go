package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// git runs a git command in dir, isolated from user/system git config.
func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-c", "user.name=T", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "HOME="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newRepo(t *testing.T) Repo {
	t.Helper()
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-q")
	return Repo{Dir: dir}
}

func (r Repo) write(t *testing.T, path, content string) {
	t.Helper()
	p := filepath.Join(r.Dir, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (r Repo) commit(t *testing.T, msg string) string {
	t.Helper()
	gitCmd(t, r.Dir, "add", "-A")
	gitCmd(t, r.Dir, "commit", "-q", "-m", msg)
	return gitCmd(t, r.Dir, "rev-parse", "HEAD")
}

func TestLog(t *testing.T) {
	r := newRepo(t)
	r.write(t, "a/x", "1")
	c1 := r.commit(t, "add x")
	r.write(t, "a/x", "2")
	r.write(t, "b/y", "y")
	c2 := r.commit(t, "second é\n\nbody line | one\n\nbody line two")
	if err := os.Remove(filepath.Join(r.Dir, "b/y")); err != nil {
		t.Fatal(err)
	}
	c3 := r.commit(t, "drop y")

	cs, err := r.Log("", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 3 || cs[0].SHA != c3 || cs[1].SHA != c2 || cs[2].SHA != c1 {
		t.Fatalf("order wrong: %+v", cs)
	}
	if !reflect.DeepEqual(cs[0].Files, []FileChange{{'D', "b/y"}}) {
		t.Errorf("c3 files: %+v", cs[0].Files)
	}
	if !reflect.DeepEqual(cs[1].Files, []FileChange{{'M', "a/x"}, {'A', "b/y"}}) {
		t.Errorf("c2 files: %+v", cs[1].Files)
	}
	if !reflect.DeepEqual(cs[2].Files, []FileChange{{'A', "a/x"}}) {
		t.Errorf("c1 files: %+v", cs[2].Files)
	}
	if cs[1].Subject != "second é" || cs[1].Body != "body line | one\n\nbody line two" {
		t.Errorf("subject/body: %q %q", cs[1].Subject, cs[1].Body)
	}
	if cs[0].Body != "" || cs[0].Author != "T" || cs[0].Email != "t@t" {
		t.Errorf("c3: %+v", cs[0])
	}
	if len(cs[2].Parents) != 0 || len(cs[0].Parents) != 1 || cs[0].Parents[0] != c2 {
		t.Errorf("parents: %+v %+v", cs[2].Parents, cs[0].Parents)
	}
	if cs[0].Short == "" || !strings.HasPrefix(c3, cs[0].Short) || cs[0].Date.IsZero() {
		t.Errorf("short/date: %+v", cs[0])
	}

	cs, err = r.Log(c1, "HEAD")
	if err != nil || len(cs) != 2 {
		t.Fatalf("range: %v %+v", err, cs)
	}
}

func TestLogMerge(t *testing.T) {
	r := newRepo(t)
	r.write(t, "a", "1")
	base := r.commit(t, "base")
	gitCmd(t, r.Dir, "checkout", "-q", "-b", "feat")
	r.write(t, "f", "f")
	fc := r.commit(t, "feat commit")
	gitCmd(t, r.Dir, "checkout", "-q", "main")
	r.write(t, "m", "m")
	mc := r.commit(t, "main commit")
	gitCmd(t, r.Dir, "merge", "-q", "--no-ff", "-m", "merge feat", "feat")

	cs, err := r.Log(base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]Commit{}
	for _, c := range cs {
		seen[c.SHA] = c
	}
	if len(cs) != 3 || seen[fc].SHA == "" || seen[mc].SHA == "" {
		t.Fatalf("commits: %+v", cs)
	}
	if cs[0].Subject != "merge feat" || len(cs[0].Parents) != 2 || cs[0].Files != nil {
		t.Errorf("merge: %+v", cs[0])
	}
	if !reflect.DeepEqual(seen[fc].Files, []FileChange{{'A', "f"}}) {
		t.Errorf("feat files: %+v", seen[fc].Files)
	}
	set, err := r.RevSet(base, "HEAD")
	if err != nil || len(set) != 3 || !set[fc] {
		t.Errorf("RevSet: %v %v", err, set)
	}
}

func TestRevParseMissing(t *testing.T) {
	r := newRepo(t)
	r.write(t, "a", "1")
	c := r.commit(t, "x")
	if _, err := r.RevParse("deadbeef"); err == nil {
		t.Error("expected error")
	}
	got, err := r.RevParse("HEAD")
	if err != nil || got != c {
		t.Errorf("HEAD: %q %v", got, err)
	}
}

func TestIsAncestor(t *testing.T) {
	r := newRepo(t)
	r.write(t, "a", "1")
	base := r.commit(t, "base")
	gitCmd(t, r.Dir, "checkout", "-q", "-b", "feat")
	r.write(t, "f", "f")
	fc := r.commit(t, "feat")
	gitCmd(t, r.Dir, "checkout", "-q", "main")
	r.write(t, "m", "m")
	mc := r.commit(t, "main")
	if ok, err := r.IsAncestor(base, mc); err != nil || !ok {
		t.Errorf("parent: %v %v", ok, err)
	}
	if ok, err := r.IsAncestor(fc, mc); err != nil || ok {
		t.Errorf("sibling: %v %v", ok, err)
	}
}

func TestTreeFiles(t *testing.T) {
	r := newRepo(t)
	r.write(t, "a/x y", "1")
	r.write(t, "b", "2")
	r.commit(t, "x")
	m, err := r.TreeFiles("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	want := gitCmd(t, r.Dir, "rev-parse", "HEAD:b")
	if len(m) != 2 || m["b"] != want || m["a/x y"] == "" {
		t.Errorf("tree: %v", m)
	}
}

func TestShowAndTracked(t *testing.T) {
	r := newRepo(t)
	r.write(t, "a/x", "hello\n")
	r.commit(t, "x")
	b, ok, err := r.Show("HEAD", "a/x")
	if err != nil || !ok || string(b) != "hello\n" {
		t.Errorf("present: %q %v %v", b, ok, err)
	}
	b, ok, err = r.Show("HEAD", "nope")
	if err != nil || ok || b != nil {
		t.Errorf("absent: %q %v %v", b, ok, err)
	}
	if ok, err := r.Tracked("a/x"); err != nil || !ok {
		t.Errorf("tracked: %v %v", ok, err)
	}
	if ok, err := r.Tracked("nope"); err != nil || ok {
		t.Errorf("untracked: %v %v", ok, err)
	}
}

func TestCommitURLTemplate(t *testing.T) {
	const want = "https://github.com/org/repo/commit/{sha}"
	for in, w := range map[string]string{
		"git@github.com:org/repo.git":            want,
		"https://github.com/org/repo":            want,
		"https://github.com/org/repo.git":        want,
		"ssh://git@github.com:2222/org/repo.git": want,
		"ssh://git@github.com/org/repo.git":      want,
		"https://host:3000/o/r.git":              "https://host:3000/o/r/commit/{sha}",
		"https://user:tok@host:3000/o/r.git":     "https://host:3000/o/r/commit/{sha}",
		"http://host:3000/o/r.git":               "http://host:3000/o/r/commit/{sha}",
		"/local/path":                            "",
		"":                                       "",
	} {
		if got := commitURLFromRemote(in); got != w {
			t.Errorf("%q: got %q want %q", in, got, w)
		}
	}
	r := newRepo(t)
	if got := r.CommitURLTemplate(); got != "" {
		t.Errorf("no remote: %q", got)
	}
	gitCmd(t, r.Dir, "remote", "add", "origin", "git@github.com:org/repo.git")
	if got := r.CommitURLTemplate(); got != want {
		t.Errorf("remote: %q", got)
	}
}

func TestSubdirDirAndHostileConfig(t *testing.T) {
	top := newRepo(t)
	gitCmd(t, top.Dir, "config", "log.showRoot", "false")
	gitCmd(t, top.Dir, "config", "log.showSignature", "true")
	gitCmd(t, top.Dir, "config", "color.ui", "always")
	top.write(t, "ws/a/x", "1")
	top.write(t, "outside/o", "o")
	c1 := top.commit(t, "root")
	top.write(t, "outside/o", "o2")
	top.commit(t, "outside only")
	top.write(t, "ws/a/x", "2")
	c3 := top.commit(t, "ws change")
	r := Repo{Dir: filepath.Join(top.Dir, "ws")}

	cs, err := r.Log("", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 3 || cs[0].SHA != c3 || cs[2].SHA != c1 {
		t.Fatalf("commits: %+v", cs)
	}
	if !reflect.DeepEqual(cs[0].Files, []FileChange{{'M', "a/x"}}) || len(cs[1].Files) != 0 ||
		!reflect.DeepEqual(cs[2].Files, []FileChange{{'A', "a/x"}}) {
		t.Errorf("files: %+v / %+v / %+v", cs[0].Files, cs[1].Files, cs[2].Files)
	}
	m, err := r.TreeFiles("HEAD")
	if err != nil || len(m) != 1 || m["a/x"] == "" {
		t.Errorf("tree: %v %v", m, err)
	}
	if ok, err := r.Tracked("a/x"); err != nil || !ok {
		t.Errorf("tracked: %v %v", ok, err)
	}
	if ok, err := r.Tracked("outside/o"); err != nil || ok {
		t.Errorf("tracked outside: %v %v", ok, err)
	}
	b, ok, err := r.Show("HEAD", "a/x")
	if err != nil || !ok || string(b) != "2" {
		t.Errorf("show: %q %v %v", b, ok, err)
	}
	if _, ok, err := r.Show("HEAD", "ws/a/x"); err != nil || ok {
		t.Errorf("show toplevel-relative should be absent: %v %v", ok, err)
	}
}

func TestDirty(t *testing.T) {
	r := newRepo(t)
	r.write(t, "a.txt", "1")
	r.commit(t, "init")
	if d, err := r.Dirty(); err != nil || d {
		t.Fatalf("clean: Dirty = %v, %v", d, err)
	}
	r.write(t, "b.txt", "2")
	if d, err := r.Dirty(); err != nil || !d {
		t.Fatalf("untracked: Dirty = %v, %v", d, err)
	}
	r.commit(t, "b")
	r.write(t, "a.txt", "changed")
	if d, err := r.Dirty(); err != nil || !d {
		t.Fatalf("modified: Dirty = %v, %v", d, err)
	}
}

func TestTagsAndTagCommit(t *testing.T) {
	r := newRepo(t)
	if tags, err := r.Tags(); err != nil || len(tags) != 0 {
		t.Fatalf("empty repo: %v, %v", tags, err)
	}
	r.write(t, "a", "1")
	c1 := r.commit(t, "one")
	r.write(t, "a", "2")
	c2 := r.commit(t, "two")
	gitCmd(t, r.Dir, "tag", "v1.0.0", c1)
	gitCmd(t, r.Dir, "tag", "-a", "-m", "annotated", "app@v2.0.0", c2)

	tags, err := r.Tags()
	if err != nil || !reflect.DeepEqual(tags, []string{"app@v2.0.0", "v1.0.0"}) {
		t.Fatalf("Tags = %v, %v", tags, err)
	}
	if sha, ok, err := r.TagCommit("v1.0.0"); err != nil || !ok || sha != c1 {
		t.Errorf("lightweight: %q %v %v want %q", sha, ok, err, c1)
	}
	// Annotated tags are peeled to the commit, not the tag object.
	if sha, ok, err := r.TagCommit("app@v2.0.0"); err != nil || !ok || sha != c2 {
		t.Errorf("annotated: %q %v %v want %q", sha, ok, err, c2)
	}
	if _, ok, err := r.TagCommit("nope"); err != nil || ok {
		t.Errorf("missing tag: ok=%v err=%v", ok, err)
	}
}

func TestParseRemote(t *testing.T) {
	tests := []struct {
		in                 string
		scheme, host, path string
		ok                 bool
	}{
		{"https://h.com/o/r.git", "https", "h.com", "o/r", true},
		{"http://u:p@h.com:3000/o/r/", "http", "h.com:3000", "o/r", true},
		{"ssh://git@h.com:2222/o/r.git", "https", "h.com", "o/r", true},
		{"git@h.com:o/r.git", "https", "h.com", "o/r", true},
		{"/local/path", "", "", "", false},
		{"https://h.com/solo", "", "", "", false},
		{"a:b@c", "", "", "", false},
	}
	for _, tt := range tests {
		s, h, p, ok := ParseRemote(tt.in)
		if s != tt.scheme || h != tt.host || p != tt.path || ok != tt.ok {
			t.Errorf("ParseRemote(%q) = %q %q %q %v", tt.in, s, h, p, ok)
		}
	}
}

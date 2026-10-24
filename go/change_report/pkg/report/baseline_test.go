package report

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
)

func gitT(t *testing.T, dir string, args ...string) string {
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

type testRepo struct {
	gitx.Repo
	t *testing.T
	n int
}

func newTestRepo(t *testing.T) *testRepo {
	dir := t.TempDir()
	gitT(t, dir, "init", "-q")
	return &testRepo{Repo: gitx.Repo{Dir: dir}, t: t}
}

func (r *testRepo) commit(msg string) string {
	r.n++
	name := filepath.Join(r.Dir, "f"+msg)
	if err := os.WriteFile(name, []byte(msg), 0o644); err != nil {
		r.t.Fatal(err)
	}
	gitT(r.t, r.Dir, "add", "-A")
	gitT(r.t, r.Dir, "commit", "-q", "-m", msg)
	return gitT(r.t, r.Dir, "rev-parse", "HEAD")
}

func TestResolveSince(t *testing.T) {
	r := newTestRepo(t)
	c1 := r.commit("one")
	r.commit("two")
	b, w := ResolveBaselines(r.Repo, "HEAD", []string{"//a:a", "//b:b"}, "HEAD~1", nil)
	if len(w) != 0 {
		t.Errorf("warnings %v", w)
	}
	want := map[string]string{"//a:a": c1, "//b:b": c1}
	if !reflect.DeepEqual(b, want) {
		t.Errorf("got %v want %v", b, want)
	}
}

func TestResolveNotAncestor(t *testing.T) {
	r := newTestRepo(t)
	r.commit("one")
	gitT(t, r.Dir, "checkout", "-q", "-b", "side")
	side := r.commit("side")
	gitT(t, r.Dir, "checkout", "-q", "main")
	r.commit("two")
	b, w := ResolveBaselines(r.Repo, "HEAD", []string{"//a:a"}, side, nil)
	if b["//a:a"] != "" {
		t.Errorf("baseline %q", b["//a:a"])
	}
	if len(w) != 1 || !strings.Contains(w[0], "is not an ancestor of HEAD; treating as changed") || !strings.Contains(w[0], "//a:a") {
		t.Errorf("warnings %v", w)
	}
}

func TestResolveUnknown(t *testing.T) {
	r := newTestRepo(t)
	r.commit("one")
	b, w := ResolveBaselines(r.Repo, "HEAD", []string{"//a:a"}, "nosuchrev", nil)
	if b["//a:a"] != "" {
		t.Errorf("baseline %q", b["//a:a"])
	}
	if len(w) != 1 || !strings.Contains(w[0], "cannot be resolved") {
		t.Errorf("warnings %v", w)
	}
}

func TestResolveEmptySince(t *testing.T) {
	r := newTestRepo(t)
	r.commit("one")
	b, w := ResolveBaselines(r.Repo, "HEAD", []string{"//a:a"}, "", nil)
	if b["//a:a"] != "" || len(w) != 0 {
		t.Errorf("got %v %v", b, w)
	}
}

func TestPerTargetOverride(t *testing.T) {
	r := newTestRepo(t)
	c1 := r.commit("one")
	c2 := r.commit("two")
	r.commit("three")
	b, w := ResolveBaselines(r.Repo, "HEAD", []string{"//a:a", "//b:b"}, "HEAD~2",
		map[string]string{"@@//b:b": "HEAD~1", "//gone:gone": "HEAD~1", "not a label": "HEAD~1"})
	want := map[string]string{"//a:a": c1, "//b:b": c2}
	if !reflect.DeepEqual(b, want) {
		t.Errorf("got %v want %v", b, want)
	}
	if len(w) != 2 {
		t.Fatalf("warnings %v", w)
	}
	joined := strings.Join(w, "\n")
	if !strings.Contains(joined, "baseline for //gone:gone ignored: not in --targets universe") {
		t.Errorf("warnings %v", w)
	}
}

func TestCollectHistoryOnePass(t *testing.T) {
	r := newTestRepo(t)
	c1 := r.commit("one")
	c2 := r.commit("two")
	c3 := r.commit("three")
	commits, inRange, err := CollectHistory(r.Repo, "HEAD", map[string]string{"//a:a": c1, "//b:b": c2, "//c:c": ""})
	if err != nil {
		t.Fatal(err)
	}
	var shas []string
	for _, c := range commits {
		shas = append(shas, c.SHA)
	}
	if !reflect.DeepEqual(shas, []string{c3, c2}) {
		t.Errorf("commits %v", shas)
	}
	if !reflect.DeepEqual(inRange[c1], map[string]bool{c2: true, c3: true}) {
		t.Errorf("inRange[c1] %v", inRange[c1])
	}
	if !reflect.DeepEqual(inRange[c2], map[string]bool{c3: true}) {
		t.Errorf("inRange[c2] %v", inRange[c2])
	}
	if len(inRange) != 2 {
		t.Errorf("inRange %v", inRange)
	}
}

func TestCollectHistoryUnrelated(t *testing.T) {
	r := newTestRepo(t)
	r.commit("one")
	gitT(t, r.Dir, "checkout", "-q", "-b", "side")
	side := r.commit("side")
	gitT(t, r.Dir, "checkout", "-q", "main")
	c2 := r.commit("two")
	commits, inRange, err := CollectHistory(r.Repo, "HEAD", map[string]string{"//a:a": side, "//b:b": c2})
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 { // full history of main: one, two
		t.Errorf("commits %d", len(commits))
	}
	if len(inRange) != 2 {
		t.Errorf("inRange %v", inRange)
	}
}

func TestCollectHistoryNoBaselines(t *testing.T) {
	r := newTestRepo(t)
	r.commit("one")
	commits, inRange, err := CollectHistory(r.Repo, "HEAD", map[string]string{"//a:a": ""})
	if err != nil || len(commits) != 0 || len(inRange) != 0 || inRange == nil {
		t.Errorf("got %v %v %v", commits, inRange, err)
	}
}

func TestResolveRootBaseline(t *testing.T) {
	r := newTestRepo(t)
	r.commit("one")
	b, w := ResolveBaselines(r.Repo, "HEAD", []string{"//a:a", "//b:b"}, RootBaseline, map[string]string{"//b:b": RootBaseline})
	if len(w) != 0 || b["//a:a"] != RootBaseline || b["//b:b"] != RootBaseline {
		t.Errorf("b=%v w=%v", b, w)
	}
}

func TestCollectHistoryRoot(t *testing.T) {
	r := newTestRepo(t)
	c1 := r.commit("one")
	c2 := r.commit("two")
	c3 := r.commit("three")
	commits, inRange, err := CollectHistory(r.Repo, "HEAD", map[string]string{"//a:a": RootBaseline, "//b:b": c2})
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 3 {
		t.Fatalf("commits %d", len(commits))
	}
	for _, c := range []string{c1, c2, c3} {
		if !inRange[RootBaseline][c] {
			t.Errorf("%s not in root range", c)
		}
	}
	if len(inRange[c2]) != 1 || !inRange[c2][c3] {
		t.Errorf("inRange[c2] = %v", inRange[c2])
	}
}

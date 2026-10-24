package report

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
)

func TestGitAccessors(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "HOME="+dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "init")

	var warns []string
	fileAt, tracked := GitAccessors(gitx.Repo{Dir: dir}, func(s string) { warns = append(warns, s) })
	if b, ok := fileAt("HEAD", "f.txt"); !ok || string(b) != "hi\n" {
		t.Errorf("fileAt(HEAD, f.txt) = %q, %v", b, ok)
	}
	if _, ok := fileAt("HEAD", "missing"); ok {
		t.Error("missing file reported present")
	}
	if !tracked("f.txt") || tracked("missing") {
		t.Error("tracked wrong")
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings %v", warns)
	}
	// A bad revision warns once and is treated as absent.
	fileAt("nosuchrev", "f.txt")
	fileAt("nosuchrev", "f.txt")
	if len(warns) != 1 || !strings.Contains(warns[0], "cannot read f.txt at nosuchrev") {
		t.Errorf("warnings %v", warns)
	}
}

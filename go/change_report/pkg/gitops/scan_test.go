package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, dir, rel, content string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-c", "user.name=T", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "HOME="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestScan(t *testing.T) {
	d := t.TempDir()
	write(t, d, "prod/a.yaml", "# rules_k8s_cd:target //a:gitops.prod\nkind: X\n")
	write(t, d, "prod/b.yml", "# rules_k8s_cd:target //a:gitops.prod\n")
	write(t, d, "dev/c.yaml", "# rules_k8s_cd:target @@//b:gitops.dev\n")
	write(t, d, "dev/none.yaml", "kind: Y\n")
	write(t, d, "dev/ignored.txt", "# rules_k8s_cd:target //z:z\n")
	write(t, d, ".git/x.yaml", "# rules_k8s_cd:target //z:z\n")
	r, err := Scan([]string{d}, "")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"//a:gitops.prod": {filepath.Join(d, "prod/a.yaml"), filepath.Join(d, "prod/b.yml")},
		"//b:gitops.dev":  {filepath.Join(d, "dev/c.yaml")},
	}
	if !reflect.DeepEqual(r.Targets, want) {
		t.Fatalf("targets = %v, want %v", r.Targets, want)
	}
	if r.Unlabeled != 1 {
		t.Fatalf("unlabeled = %d", r.Unlabeled)
	}
}

func TestScanBadLabelUnlabeled(t *testing.T) {
	d := t.TempDir()
	write(t, d, "a.yaml", "# rules_k8s_cd:target not a label\n")
	r, err := Scan([]string{d}, "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Unlabeled != 1 || len(r.Targets) != 0 {
		t.Fatalf("got %+v", r)
	}
}

func TestScanCRLF(t *testing.T) {
	d := t.TempDir()
	write(t, d, "a.yaml", "# rules_k8s_cd:target //a:b\r\nkind: X\r\n")
	r, err := Scan([]string{d}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Targets["//a:b"]) != 1 {
		t.Fatalf("got %+v", r)
	}
}

func TestScanCache(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 000 does not block root")
	}
	d := t.TempDir()
	cf := filepath.Join(t.TempDir(), "sub", "cache.json")
	p := write(t, d, "a.yaml", "# rules_k8s_cd:target //a:b\n")
	if _, err := Scan([]string{d}, cf); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(p, 0o644)
	r, err := Scan([]string{d}, cf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Targets["//a:b"], []string{p}) {
		t.Fatalf("got %+v", r)
	}
}

func TestScanCorruptCache(t *testing.T) {
	d := t.TempDir()
	cf := filepath.Join(t.TempDir(), "cache.json")
	os.WriteFile(cf, []byte("{garbage"), 0o644)
	write(t, d, "a.yaml", "# rules_k8s_cd:target //a:b\n")
	r, err := Scan([]string{d}, cf)
	if err != nil || len(r.Targets["//a:b"]) != 1 {
		t.Fatalf("got %+v, %v", r, err)
	}
}

func TestScanGitBlobIdentity(t *testing.T) {
	root := t.TempDir()
	gitCmd(t, root, "init", "-q")
	d := filepath.Join(root, "out")
	p := write(t, d, "a.yaml", "# rules_k8s_cd:target //a:one\n")
	write(t, d, "b.yaml", "# rules_k8s_cd:target //a:two\n")
	gitCmd(t, root, "add", "-A")
	gitCmd(t, root, "commit", "-q", "-m", "x")
	cf := filepath.Join(t.TempDir(), "cache.json")
	r, err := Scan([]string{d}, cf)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Targets["//a:one"]) != 1 || len(r.Targets["//a:two"]) != 1 {
		t.Fatalf("got %+v", r)
	}
	// Uncommitted modification of the same length must not return a stale label.
	write(t, d, "a.yaml", "# rules_k8s_cd:target //a:six\n")
	os.Chtimes(p, mustStat(t, filepath.Join(d, "b.yaml")), mustStat(t, filepath.Join(d, "b.yaml")))
	r, err = Scan([]string{d}, cf)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Targets["//a:six"]) != 1 || len(r.Targets["//a:one"]) != 0 {
		t.Fatalf("stale label: %+v", r)
	}
	// Untracked file is scanned too.
	write(t, d, "new.yaml", "# rules_k8s_cd:target //n:n\n")
	r, _ = Scan([]string{d}, cf)
	if len(r.Targets["//n:n"]) != 1 {
		t.Fatalf("untracked missing: %+v", r)
	}
	if strings.Contains(strings.Join(r.Targets["//a:two"], ""), "..") {
		t.Fatal("bad path")
	}
}

func mustStat(t *testing.T, p string) (tm time.Time) {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime()
}

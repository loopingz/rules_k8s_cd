package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitops"
)

func runCLI(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb, func(string) string { return "" })
	return code, out.String(), errb.String()
}

func TestFlagErrors(t *testing.T) {
	if code, _, _ := runCLI(); code != 2 {
		t.Errorf("missing --targets: code = %d, want 2", code)
	}
	if code, _, _ := runCLI("--targets", "//...", "--format", "bogus"); code != 2 {
		t.Errorf("bad --format: code = %d, want 2", code)
	}
	if code, _, _ := runCLI("--nope"); code != 2 {
		t.Errorf("unknown flag: code = %d, want 2", code)
	}
}

func TestSinceFileParse(t *testing.T) {
	f := filepath.Join(t.TempDir(), "since.json")
	if err := os.WriteFile(f, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCLI("--targets", "//...", "--since-file", f)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "error: --since-file:") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestRelativeSinceFileUsesWorkingDirectory(t *testing.T) {
	wd := t.TempDir()
	if err := os.WriteFile(filepath.Join(wd, "since.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	env := func(k string) string {
		if k == "BUILD_WORKING_DIRECTORY" {
			return wd
		}
		return ""
	}
	// Found under wd: fails on JSON parsing, not on a missing file.
	code := run([]string{"--targets", "//...", "--since-file", "since.json"}, &out, &errb, env)
	if code != 1 || !strings.Contains(errb.String(), "invalid character") {
		t.Errorf("code=%d stderr=%q", code, errb.String())
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out.txt")
	if err := writeFileAtomic(p, []byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(p, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "new" {
		t.Errorf("content %q", b)
	}
	// Failure (target is a non-empty directory) leaves no temp file behind.
	d := filepath.Join(dir, "d")
	os.Mkdir(d, 0o755)
	os.WriteFile(filepath.Join(d, "x"), nil, 0o644)
	if err := writeFileAtomic(d, []byte("z")); err == nil {
		t.Error("want error")
	}
	// Failing run leaves an existing --output untouched.
	os.WriteFile(p, []byte("keep"), 0o644)
	if code, _, _ := runCLI("--targets", "//...", "--since-file", filepath.Join(dir, "missing.json"), "--output", p); code != 1 {
		t.Errorf("code=%d", code)
	}
	if b, _ := os.ReadFile(p); string(b) != "keep" {
		t.Errorf("output clobbered: %q", b)
	}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover %s", e.Name())
		}
	}
}

type fakeBazel struct {
	out   string
	code  int // exit code; 0 = success
	calls [][]string
}

type fakeExit int

func (e fakeExit) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e fakeExit) ExitCode() int { return int(e) }

func (f *fakeBazel) Run(args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if f.code != 0 {
		return []byte(f.out), fmt.Errorf("bazel: %w", fakeExit(f.code))
	}
	return []byte(f.out), nil
}

func TestExistingOutsideUniverse(t *testing.T) {
	gres := &gitops.Result{Targets: map[string][]string{
		"//in:u":   {"u.yaml"},
		"//live:x": {"x.yaml"},
		"//gone:y": {"y.yaml"},
	}}
	universe := []string{"//in:u"}
	var warns []string
	warn := func(s string) { warns = append(warns, s) }

	f := &fakeBazel{out: "@@//live:x\n", code: 3}
	got := existingOutsideUniverse(f, gres, universe, warn)
	if want := map[string]bool{"//live:x": true}; !reflect.DeepEqual(got, want) {
		t.Errorf("exit 3: got %v, want %v", got, want)
	}
	if q := f.calls[0][len(f.calls[0])-1]; q != `set("//gone:y" "//live:x")` {
		t.Errorf("query %q", q)
	}
	if len(warns) != 1 || warns[0] != "1 gitops target(s) outside --targets still exist and were not marked removed" {
		t.Errorf("warnings %v", warns)
	}

	warns = nil
	got = existingOutsideUniverse(&fakeBazel{code: 2}, gres, universe, warn)
	if want := map[string]bool{"//live:x": true, "//gone:y": true}; !reflect.DeepEqual(got, want) {
		t.Errorf("query failure: got %v, want every candidate", got)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "none marked removed") {
		t.Errorf("warnings %v", warns)
	}

	f = &fakeBazel{}
	if got := existingOutsideUniverse(f, &gitops.Result{Targets: map[string][]string{"//in:u": nil}}, universe, warn); got != nil || len(f.calls) != 0 {
		t.Errorf("no candidates: got %v, calls %v", got, f.calls)
	}
}

func TestDisplayPath(t *testing.T) {
	for _, c := range []struct{ base, p, want string }{
		{"/work", "/work/deploy/a.yaml", "deploy/a.yaml"},
		{"/work/", "/work/a.yaml", "a.yaml"},
		{"/work", "/workx/a.yaml", "/workx/a.yaml"},
		{"/work/sub", "/work/a.yaml", "/work/a.yaml"},
		{"", "/work/a.yaml", "/work/a.yaml"},
	} {
		if got := displayPath(c.base, c.p); got != c.want {
			t.Errorf("displayPath(%q, %q) = %q, want %q", c.base, c.p, got, c.want)
		}
	}
}

func TestWorkingDir(t *testing.T) {
	if got := workingDir(func(k string) string {
		if k == "BUILD_WORKING_DIRECTORY" {
			return "/from/bazel"
		}
		return ""
	}); got != "/from/bazel" {
		t.Errorf("got %q", got)
	}
	wd, _ := os.Getwd()
	if got := workingDir(func(string) string { return "" }); got != wd {
		t.Errorf("got %q, want cwd %q", got, wd)
	}
}

func TestIgnoreFlag(t *testing.T) {
	f := ignorePathFunc([]string{"docs/**", "*.md"})
	if f == nil || !f("//a:a", "docs/x/y.txt") || !f("//a:a", "README.md") || f("//a:a", "src/a.go") {
		t.Error("ignore globs not applied")
	}
	if ignorePathFunc(nil) != nil {
		t.Error("no globs must give nil")
	}
	_, _, stderr := runCLI("--targets", "//...", "--ignore", "docs/**", "--ignore", "*.md", "--format", "bogus")
	if strings.Contains(stderr, "flag provided but not defined") || strings.Contains(stderr, "invalid value") {
		t.Errorf("--ignore rejected: %s", stderr)
	}
	if !strings.Contains(stderr, "invalid --format") {
		t.Errorf("flags not parsed up to --format validation: %s", stderr)
	}
}

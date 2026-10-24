package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestResolvePath(t *testing.T) {
	env := func(k string) string {
		if k == "BUILD_WORKING_DIRECTORY" {
			return "/wd"
		}
		return ""
	}
	if got := resolvePath(env, "a/b"); got != "/wd/a/b" {
		t.Errorf("relative: %q", got)
	}
	if got := resolvePath(env, "/abs"); got != "/abs" {
		t.Errorf("absolute: %q", got)
	}
	if got := resolvePath(func(string) string { return "" }, "a"); got != "a" {
		t.Errorf("no wd: %q", got)
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

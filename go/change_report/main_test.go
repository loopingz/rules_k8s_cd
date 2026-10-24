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

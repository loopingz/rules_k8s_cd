package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// makeTarGz creates a .tar.gz file at path with the given entries (name -> content).
func makeTarGz(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range entries {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write content %s: %v", name, err)
		}
	}
}

// readTarGz returns a map of name -> content for all entries in the .tar.gz at path.
func readTarGz(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open tar.gz: %v", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	result := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		buf := make([]byte, hdr.Size)
		if hdr.Size > 0 {
			if _, err := io.ReadFull(tr, buf); err != nil && err != io.EOF {
				t.Fatalf("read content %s: %v", hdr.Name, err)
			}
		}
		result[hdr.Name] = string(buf)
	}
	return result
}

// resetGlobals resets all package-level flags to their zero values between tests.
func resetGlobals() {
	output = ""
	inputs = arrayFlags{}
	filters = regexFlags{}
	substitutions = Substitutions{}
	verbose = false
}

func TestBasicFiltering(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "input.tar.gz")
	out := filepath.Join(dir, "output.tar.gz")

	makeTarGz(t, in, map[string]string{
		"app.go":    "package main",
		"debug.log": "some log",
		"README.md": "readme",
	})

	resetGlobals()
	output = out
	inputs = arrayFlags{in}
	r, err := regexp.Compile(`\.log$`)
	if err != nil {
		t.Fatal(err)
	}
	filters = regexFlags{r}

	process()

	got := readTarGz(t, out)
	if _, ok := got["debug.log"]; ok {
		t.Error("debug.log should have been filtered out")
	}
	if _, ok := got["app.go"]; !ok {
		t.Error("app.go should be present")
	}
	if _, ok := got["README.md"]; !ok {
		t.Error("README.md should be present")
	}
}

func TestPathSubstitution(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "input.tar.gz")
	out := filepath.Join(dir, "output.tar.gz")

	makeTarGz(t, in, map[string]string{
		"old/file.txt": "content",
		"old/other.go": "package main",
	})

	resetGlobals()
	output = out
	inputs = arrayFlags{in}
	r, err := regexp.Compile(`^old/`)
	if err != nil {
		t.Fatal(err)
	}
	substitutions = Substitutions{{regex: r, new: "new/"}}

	process()

	got := readTarGz(t, out)
	if _, ok := got["old/file.txt"]; ok {
		t.Error("old/file.txt should have been substituted")
	}
	if _, ok := got["new/file.txt"]; !ok {
		t.Error("new/file.txt should be present after substitution")
	}
	if _, ok := got["new/other.go"]; !ok {
		t.Error("new/other.go should be present after substitution")
	}
}

func TestEmptyArchive(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "empty.tar.gz")
	out := filepath.Join(dir, "output.tar.gz")

	makeTarGz(t, in, map[string]string{})

	resetGlobals()
	output = out
	inputs = arrayFlags{in}

	process()

	got := readTarGz(t, out)
	if len(got) != 0 {
		t.Errorf("expected empty output archive, got %d entries", len(got))
	}
}

func TestDuplicateDedup(t *testing.T) {
	dir := t.TempDir()
	in1 := filepath.Join(dir, "first.tar.gz")
	in2 := filepath.Join(dir, "second.tar.gz")
	out := filepath.Join(dir, "output.tar.gz")

	makeTarGz(t, in1, map[string]string{
		"shared.txt":  "from first",
		"unique1.txt": "only in first",
	})
	makeTarGz(t, in2, map[string]string{
		"shared.txt":  "from second",
		"unique2.txt": "only in second",
	})

	resetGlobals()
	output = out
	inputs = arrayFlags{in1, in2}

	process()

	got := readTarGz(t, out)
	if got["shared.txt"] != "from first" {
		t.Errorf("expected first occurrence to win, got %q", got["shared.txt"])
	}
	if _, ok := got["unique1.txt"]; !ok {
		t.Error("unique1.txt should be present")
	}
	if _, ok := got["unique2.txt"]; !ok {
		t.Error("unique2.txt should be present")
	}
	if len(got) != 3 {
		t.Errorf("expected 3 entries, got %d", len(got))
	}
}

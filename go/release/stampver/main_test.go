package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct {
	t   *testing.T
	dir string
}

func newFixture(t *testing.T) *fixture { return &fixture{t: t, dir: t.TempDir()} }

func (f *fixture) write(name, content string) string {
	p := filepath.Join(f.dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return p
}

type result struct {
	Version    string `json:"version"`
	Commit     string `json:"commit"`
	CommitFull string `json:"commit_full"`
	Date       string `json:"date"`
	DateISO    string `json:"date_iso"`
	Component  string `json:"component"`
	Tagged     bool   `json:"tagged"`
}

// version runs `stampver version` and returns the txt output and parsed JSON.
func (f *fixture) version(extra ...string) (string, result) {
	f.t.Helper()
	txt := filepath.Join(f.dir, "out.txt")
	js := filepath.Join(f.dir, "out.json")
	args := append([]string{"version", "--component", "web", "--tag-format", "{name}@v{version}",
		"--untagged", "main-{commit}", "--out", txt, "--out-json", js}, extra...)
	var so, se bytes.Buffer
	if code := run(args, &so, &se); code != 0 {
		f.t.Fatalf("run = %d: %s", code, se.String())
	}
	b, _ := os.ReadFile(txt)
	jb, _ := os.ReadFile(js)
	var r result
	if err := json.Unmarshal(jb, &r); err != nil {
		f.t.Fatal(err)
	}
	return string(b), r
}

func TestEmbedLabelWins(t *testing.T) {
	f := newFixture(t)
	vf := f.write("VERSION", "1.2.3\n")
	st := f.write("stable", "STABLE_GIT_COMMIT abc1234\nBUILD_EMBED_LABEL web@v1.2.3\n")
	txt, r := f.version("--version-file", vf, "--stable-status", st)
	if txt != "1.2.3\n" || !r.Tagged || r.Version != "1.2.3" || r.Commit != "abc1234" || r.Component != "web" {
		t.Errorf("got %q %+v", txt, r)
	}
}

func TestTagAtHead(t *testing.T) {
	f := newFixture(t)
	vf := f.write("VERSION", "1.2.3\n")
	st := f.write("stable", "STABLE_GIT_COMMIT abc1234\nSTABLE_GIT_TAGS_AT_HEAD other@v1.0.0 web@v1.2.3\n")
	txt, r := f.version("--version-file", vf, "--stable-status", st)
	if txt != "1.2.3\n" || !r.Tagged {
		t.Errorf("got %q %+v", txt, r)
	}
}

func TestTagForOtherVersionIsUntagged(t *testing.T) {
	f := newFixture(t)
	vf := f.write("VERSION", "1.2.4\n")
	st := f.write("stable", "STABLE_GIT_COMMIT abc1234\nSTABLE_GIT_TAGS_AT_HEAD web@v1.2.3\n")
	txt, r := f.version("--version-file", vf, "--stable-status", st)
	if txt != "main-abc1234\n" || r.Tagged {
		t.Errorf("got %q %+v", txt, r)
	}
}

func TestUntaggedTemplate(t *testing.T) {
	f := newFixture(t)
	vf := f.write("VERSION", "1.2.3\n")
	st := f.write("stable", "STABLE_GIT_COMMIT abc1234\nSTABLE_GIT_COMMIT_FULL abc1234ffff\nSTABLE_GIT_COMMIT_DATE 2026-10-24T10:00:00+02:00\nSTABLE_GIT_BRANCH main\n")
	txt, r := f.version("--version-file", vf, "--stable-status", st)
	if txt != "main-abc1234\n" || r.Tagged || r.Version != "main-abc1234" {
		t.Errorf("got %q %+v", txt, r)
	}
	if r.Date != "20261024" || r.DateISO != "2026-10-24T10:00:00+02:00" || r.CommitFull != "abc1234ffff" {
		t.Errorf("got %+v", r)
	}
	f2 := newFixture(t)
	st2 := f2.write("stable", st2Content)
	txt, _ = f2.version("--stable-status", st2, "--untagged", "{component}-{branch}-{date}-{commit_full}")
	if txt != "web-main-20261024-abc1234ffff\n" {
		t.Errorf("got %q", txt)
	}
}

const st2Content = "STABLE_GIT_COMMIT abc1234\nSTABLE_GIT_COMMIT_FULL abc1234ffff\nSTABLE_GIT_COMMIT_DATE 2026-10-24T10:00:00+02:00\nSTABLE_GIT_BRANCH main\n"

func TestNoStatus(t *testing.T) {
	f := newFixture(t)
	txt, r := f.version("--version-file", filepath.Join(f.dir, "missing"))
	if txt != "main-unknown\n" || r.Tagged || r.Date != "unknown" || r.Commit != "unknown" {
		t.Errorf("got %q %+v", txt, r)
	}
}

func TestBadDateIsUnknown(t *testing.T) {
	f := newFixture(t)
	st := f.write("stable", "STABLE_GIT_COMMIT_DATE garbage\n")
	_, r := f.version("--stable-status", st)
	if r.Date != "unknown" {
		t.Errorf("date = %q", r.Date)
	}
}

func TestStatusEmptyValueAndVolatile(t *testing.T) {
	f := newFixture(t)
	st := f.write("stable", "STABLE_GIT_COMMIT\nSTABLE_GIT_BRANCH feat/x y\n")
	vol := f.write("volatile", "BUILD_TIMESTAMP 123\n")
	txt, r := f.version("--stable-status", st, "--volatile-status", vol)
	if txt != "main-unknown\n" || r.Commit != "unknown" {
		t.Errorf("got %q %+v", txt, r)
	}
}

func TestVersionFileNeededOnlyForComponentTags(t *testing.T) {
	f := newFixture(t)
	// No VERSION and no tag of this component: untagged, not an error.
	other := f.write("other", "BUILD_EMBED_LABEL api@v1.2.3\nSTABLE_GIT_TAGS_AT_HEAD api@v1.2.3\n")
	txt, r := f.version("--stable-status", other)
	if r.Tagged || txt != "main-unknown\n" {
		t.Errorf("got %q %+v", txt, r)
	}
	// A tag of this component at HEAD makes VERSION required.
	mine := f.write("mine", "STABLE_GIT_TAGS_AT_HEAD web@v1.2.3\n")
	bad := f.write("VERSION", "nope\n")
	for _, vf := range []string{bad, filepath.Join(f.dir, "missing")} {
		var so, se bytes.Buffer
		args := []string{"version", "--component", "web", "--tag-format", "{name}@v{version}", "--untagged", "x",
			"--version-file", vf, "--stable-status", mine, "--out", filepath.Join(f.dir, "a"), "--out-json", filepath.Join(f.dir, "b")}
		if code := run(args, &so, &se); code == 0 {
			t.Errorf("version file %s: expected failure", vf)
		}
	}
}

func TestRender(t *testing.T) {
	f := newFixture(t)
	info := f.write("info.json", `{"version":"1.2.3","commit":"abc1234","commit_full":"abc1234ffff","date":"20261024","component":"web","tagged":true}`)
	tmpl := f.write("version.go.tpl", "package {package}\nconst V = \"{version}\" // {commit} {commit_full} {date} {component}\nfunc f() { x := map[string]int{}; _ = {unknown} }\n")
	out := filepath.Join(f.dir, "version.go")
	var so, se bytes.Buffer
	code := run([]string{"render", "--info", info, "--template", tmpl, "--out", out, "--sub", "package=website"}, &so, &se)
	if code != 0 {
		t.Fatalf("code %d: %s", code, se.String())
	}
	b, _ := os.ReadFile(out)
	want := "package website\nconst V = \"1.2.3\" // abc1234 abc1234ffff 20261024 web\nfunc f() { x := map[string]int{}; _ = {unknown} }\n"
	if string(b) != want {
		t.Errorf("got:\n%s\nwant:\n%s", b, want)
	}
}

func TestUsageErrors(t *testing.T) {
	var so, se bytes.Buffer
	for _, args := range [][]string{{}, {"bogus"}, {"version"}, {"render"}, {"render", "--sub", "bad"}} {
		if code := run(args, &so, &se); code == 0 {
			t.Errorf("run(%v) = 0", args)
		}
	}
	if !strings.Contains(se.String(), "stampver") {
		t.Errorf("stderr lacks usage: %q", se.String())
	}
}

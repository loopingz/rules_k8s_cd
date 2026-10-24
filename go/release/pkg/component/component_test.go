package component

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

type fakeRunner struct {
	out  string
	args []string
}

func (f *fakeRunner) Run(args ...string) ([]byte, error) {
	f.args = args
	return []byte(f.out), nil
}

func fixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/components.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDiscover(t *testing.T) {
	f := &fakeRunner{out: fixture(t)}
	got, err := Discover(f, "")
	if err != nil {
		t.Fatal(err)
	}
	if q := f.args[len(f.args)-1]; q != `kind("_release_component rule", //...)` {
		t.Errorf("query = %q", q)
	}
	want := []Component{
		{
			Label: "//lib/tests/release:demo", Name: "demo", Package: "lib/tests/release",
			Targets:     []string{"//lib/tests/release:demo_files"},
			VersionFile: "lib/tests/release/VERSION", Changelog: "lib/tests/release/CHANGELOG.md",
			Untagged: "main-{commit}", TagFormat: "{name}@v{version}",
			Initial: semver.Version{Minor: 1},
		},
		{
			Label: "//lib/tests/release:fresh", Name: "fresh", Package: "lib/tests/release",
			TargetsQuery: "//lib/tests/release:demo_files",
			VersionFile:  "lib/tests/release/FRESH_VERSION", Changelog: "lib/tests/release/FRESH_CHANGELOG.md",
			Untagged: "dev-{component}", TagFormat: "fresh-v{version}",
			Initial:        semver.Version{Minor: 1},
			Ignore:         []string{"lib/tests/release/data.txt"},
			BumpFiles:      map[string]string{"lib/tests/release/data.txt": "^(hello)$"},
			ReleaseTargets: []string{"//lib/tests/release:demo_files"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	if e := got[0].UniverseExpr(); e != `set("//lib/tests/release:demo_files")` {
		t.Errorf("UniverseExpr = %q", e)
	}
	if e := got[1].UniverseExpr(); e != `(//lib/tests/release:demo_files)` {
		t.Errorf("UniverseExpr = %q", e)
	}
}

func TestDiscoverScope(t *testing.T) {
	f := &fakeRunner{out: fixture(t)}
	if _, err := Discover(f, "//lib/..."); err != nil {
		t.Fatal(err)
	}
	if q := f.args[len(f.args)-1]; q != `kind("_release_component rule", //lib/...)` {
		t.Errorf("query = %q", q)
	}
}

func TestDiscoverDuplicate(t *testing.T) {
	lines := strings.Split(strings.TrimSpace(fixture(t)), "\n")
	other := strings.ReplaceAll(lines[0], "//lib/tests/release:demo", "//other:demo")
	_, err := Discover(&fakeRunner{out: fixture(t) + other + "\n"}, "")
	if err == nil || !strings.Contains(err.Error(), "duplicate") || !strings.Contains(err.Error(), "demo") {
		t.Errorf("err = %v", err)
	}
}

func TestDiscoverRootPackage(t *testing.T) {
	out := strings.NewReplacer("@@//lib/tests/release:", "@@//:", "lib/tests/release/", "").Replace(fixture(t))
	got, err := Discover(&fakeRunner{out: out}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got[1].Package != "" || got[1].Label != "//:fresh" || got[1].VersionFile != "FRESH_VERSION" ||
		!reflect.DeepEqual(got[1].Ignore, []string{"data.txt"}) || got[1].BumpFiles["data.txt"] == "" {
		t.Errorf("root package: %+v", got[1])
	}
}

func TestTagParseTag(t *testing.T) {
	v := semver.Version{Major: 1, Minor: 2, Patch: 3}
	c := Component{Name: "api", TagFormat: "{name}@v{version}"}
	if got := c.Tag(v); got != "api@v1.2.3" {
		t.Errorf("Tag = %q", got)
	}
	if p, ok := c.ParseTag("api@v1.2.3"); !ok || p != v {
		t.Errorf("ParseTag = %v %v", p, ok)
	}
	for _, bad := range []string{"web@v1.2.3", "v1.2.3", "api@v1.2", "api@v1.2.3-rc1"} {
		if _, ok := c.ParseTag(bad); ok {
			t.Errorf("ParseTag(%q) should fail", bad)
		}
	}
	c = Component{Name: "api", TagFormat: "v{version}"}
	if got := c.Tag(v); got != "v1.2.3" {
		t.Errorf("Tag = %q", got)
	}
	if p, ok := c.ParseTag("v4.1.0"); !ok || p.String() != "4.1.0" {
		t.Errorf("ParseTag = %v %v", p, ok)
	}
}

// --- Current ---

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

func commit(t *testing.T, dir, file, content string) string {
	t.Helper()
	p := filepath.Join(dir, file)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", "-A")
	gitCmd(t, dir, "commit", "-q", "-m", "c "+content)
	return gitCmd(t, dir, "rev-parse", "HEAD")
}

func newRepo(t *testing.T) (gitx.Repo, string) {
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-q")
	return gitx.Repo{Dir: dir}, dir
}

func TestCurrentVersionFileAndTag(t *testing.T) {
	repo, dir := newRepo(t)
	c1 := commit(t, dir, "pkg/VERSION", "1.2.0\n")
	gitCmd(t, dir, "tag", "api@v1.2.0")
	commit(t, dir, "x", "x")
	c := Component{Name: "api", TagFormat: "{name}@v{version}", VersionFile: "pkg/VERSION"}
	tags, err := ListTags(repo)
	if err != nil {
		t.Fatal(err)
	}
	st, warns, err := Current(repo, c, tags)
	if err != nil || len(warns) != 0 {
		t.Fatalf("%v %v", warns, err)
	}
	if st.Current == nil || st.Current.String() != "1.2.0" || st.Source != "version_file" || st.BaselineSHA != c1 {
		t.Errorf("state = %+v (%v)", st, st.Current)
	}
}

func TestCurrentAnnotatedTagPeeled(t *testing.T) {
	repo, dir := newRepo(t)
	c1 := commit(t, dir, "VERSION", "1.0.0\n")
	gitCmd(t, dir, "tag", "-a", "-m", "rel", "api@v1.0.0")
	tags, _ := ListTags(repo)
	st, _, err := Current(repo, Component{Name: "api", TagFormat: "{name}@v{version}", VersionFile: "VERSION"}, tags)
	if err != nil || st.BaselineSHA != c1 {
		t.Errorf("%+v %v want %s", st, err, c1)
	}
}

func TestCurrentFromTagsOnly(t *testing.T) {
	repo, dir := newRepo(t)
	commit(t, dir, "a", "1")
	gitCmd(t, dir, "tag", "v4.0.3")
	c410 := commit(t, dir, "a", "2")
	gitCmd(t, dir, "tag", "v4.1.0")
	gitCmd(t, dir, "tag", "v4.0.10") // sorts between 4.0.3 and 4.1.0 lexically, but is lower than 4.1.0
	gitCmd(t, dir, "tag", "other@v9.9.9")
	commit(t, dir, "a", "3")
	tags, _ := ListTags(repo)
	c := Component{Name: "api", TagFormat: "v{version}", VersionFile: "VERSION"}
	st, warns, err := Current(repo, c, tags)
	if err != nil || len(warns) != 0 {
		t.Fatalf("%v %v", warns, err)
	}
	if st.Current == nil || st.Current.String() != "4.1.0" || st.Source != "tag" || st.BaselineSHA != c410 {
		t.Errorf("state = %+v", st)
	}
}

func TestCurrentHighestIsSemverNotLexical(t *testing.T) {
	repo, dir := newRepo(t)
	commit(t, dir, "a", "1")
	gitCmd(t, dir, "tag", "v1.9.0")
	c := commit(t, dir, "a", "2")
	gitCmd(t, dir, "tag", "v1.10.0")
	tags, _ := ListTags(repo)
	st, _, err := Current(repo, Component{Name: "api", TagFormat: "v{version}"}, tags)
	if err != nil || st.Current.String() != "1.10.0" || st.BaselineSHA != c {
		t.Errorf("%+v %v", st, err)
	}
}

func TestCurrentNothing(t *testing.T) {
	repo, dir := newRepo(t)
	commit(t, dir, "a", "1")
	tags, _ := ListTags(repo)
	st, warns, err := Current(repo, Component{Name: "api", TagFormat: "{name}@v{version}", VersionFile: "VERSION"}, tags)
	if err != nil || len(warns) != 0 {
		t.Fatalf("%v %v", warns, err)
	}
	if st.Current != nil || st.Source != "none" || st.BaselineSHA != report.RootBaseline {
		t.Errorf("state = %+v", st)
	}
}

func TestCurrentVersionWithoutTag(t *testing.T) {
	repo, dir := newRepo(t)
	commit(t, dir, "a", "1")
	c1 := commit(t, dir, "VERSION", "2.0.0\n")
	c := Component{Name: "api", TagFormat: "{name}@v{version}", VersionFile: "VERSION"}

	// No tag at all: warning, root baseline.
	tags, _ := ListTags(repo)
	st, warns, err := Current(repo, c, tags)
	if err != nil {
		t.Fatal(err)
	}
	wantW := []string{"VERSION 2.0.0 of api has no tag api@v2.0.0; using latest tag as baseline"}
	if !reflect.DeepEqual(warns, wantW) {
		t.Errorf("warnings = %q", warns)
	}
	if st.Current == nil || st.Current.String() != "2.0.0" || st.Source != "version_file" || st.BaselineSHA != report.RootBaseline {
		t.Errorf("state = %+v", st)
	}

	// An older tag exists: it is the baseline.
	gitCmd(t, dir, "tag", "api@v1.0.0", c1)
	tags, _ = ListTags(repo)
	st, warns, err = Current(repo, c, tags)
	if err != nil || len(warns) != 1 || st.BaselineSHA != c1 || st.Current.String() != "2.0.0" {
		t.Errorf("state = %+v warns=%v err=%v", st, warns, err)
	}
}

func TestCurrentInvalidVersionFile(t *testing.T) {
	repo, dir := newRepo(t)
	commit(t, dir, "VERSION", "banana\n")
	_, _, err := Current(repo, Component{Name: "api", TagFormat: "v{version}", VersionFile: "VERSION"}, nil)
	if err == nil || !strings.Contains(err.Error(), "VERSION") {
		t.Errorf("err = %v", err)
	}
}

func TestUniverseExprQuotesLabels(t *testing.T) {
	c := Component{Targets: []string{"//a:x+y", "//c:d=e,f#g"}}
	if got := c.UniverseExpr(); got != `set("//a:x+y" "//c:d=e,f#g")` {
		t.Errorf("got %q", got)
	}
}

func TestDiscoverRejectsQuoteInTarget(t *testing.T) {
	out := strings.Replace(fixture(t), "demo_files\"]", "de\\\"mo\"]", 1)
	if out == fixture(t) {
		t.Fatal("fixture replace failed")
	}
	if _, err := Discover(&fakeRunner{out: out}, ""); err == nil {
		t.Error("expected error for label containing a quote")
	}
}

type errRunner struct{ err error }

func (e errRunner) Run(...string) ([]byte, error) { return nil, e.err }

type exitErr struct{}

func (exitErr) Error() string { return "exit status 3" }
func (exitErr) ExitCode() int { return 3 }

func TestDiscoverPropagatesQueryErrors(t *testing.T) {
	for _, err := range []error{errors.New("boom"), fmt.Errorf("bazel query: %w: partial", exitErr{})} {
		got, derr := Discover(errRunner{err}, "")
		if derr == nil || got != nil {
			t.Errorf("err %v: got %v, %v", err, got, derr)
		}
	}
}

func TestCurrentVersionBelowLatestTag(t *testing.T) {
	repo, dir := newRepo(t)
	commit(t, dir, "VERSION", "4.0.0\n")
	gitCmd(t, dir, "tag", "v4.0.0")
	c41 := commit(t, dir, "a", "2")
	gitCmd(t, dir, "tag", "v4.1.0")
	tags, _ := ListTags(repo)
	st, warns, err := Current(repo, Component{Name: "api", TagFormat: "v{version}", VersionFile: "VERSION"}, tags)
	if err != nil {
		t.Fatal(err)
	}
	if st.Current == nil || st.Current.String() != "4.1.0" || st.BaselineSHA != c41 {
		t.Errorf("state = %+v", st)
	}
	if !reflect.DeepEqual(warns, []string{"VERSION 4.0.0 of api is below the latest tag v4.1.0; using v4.1.0"}) {
		t.Errorf("warnings = %q", warns)
	}
}

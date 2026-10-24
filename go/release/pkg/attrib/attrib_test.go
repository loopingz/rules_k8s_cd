package attrib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
)

// --- fake bazel ---

// fakeRunner answers `--output=label` queries from labels (keyed by query
// expression) and every streamed_jsonproto query with nodes.
type fakeRunner struct {
	labels map[string][]string
	nodes  []string
	calls  []string
}

func (f *fakeRunner) Run(args ...string) ([]byte, error) {
	expr := args[len(args)-1]
	f.calls = append(f.calls, expr)
	switch {
	case has(args, "--output=label"):
		ls, ok := f.labels[expr]
		if !ok {
			return nil, fmt.Errorf("fake: unexpected label query %q", expr)
		}
		return []byte(strings.Join(ls, "\n") + "\n"), nil
	case has(args, "--output=streamed_jsonproto"):
		return []byte(strings.Join(f.nodes, "\n") + "\n"), nil
	}
	return nil, fmt.Errorf("fake: unexpected call %v", args)
}

func has(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func src(l string) string { return fmt.Sprintf(`{"type":"SOURCE_FILE","sourceFile":{"name":%q}}`, l) }

func rule(l string, inputs ...string) string {
	q := make([]string, len(inputs))
	for i, in := range inputs {
		q[i] = fmt.Sprintf("%q", in)
	}
	return fmt.Sprintf(`{"type":"RULE","rule":{"name":%q,"ruleClass":"filegroup","ruleInput":[%s]}}`, l, strings.Join(q, ","))
}

// deps is the graph of the fixture: //a:t and //b:t both depend on //lib:l;
// //b:t also depends on a's VERSION through the //a:version filegroup.
var deps = []string{
	rule("@@//a:t", "@@//a:src.txt", "@@//a:bump.txt", "@@//a:notes.md", "@@//lib:l"),
	rule("@@//b:t", "@@//b:src.txt", "@@//lib:l", "@@//a:version"),
	rule("@@//lib:l", "@@//lib:x"),
	rule("@@//a:version", "@@//a:VERSION"),
	src("@@//a:src.txt"), src("@@//a:bump.txt"), src("@@//a:notes.md"), src("@@//a:VERSION"),
	src("@@//b:src.txt"), src("@@//lib:x"),
}

const universe = `set("//a:t") + set("//b:t")`

// --- git fixture ---

type fixture struct {
	t     *testing.T
	dir   string
	repo  gitx.Repo
	run   *fakeRunner
	comps []component.Component
	base  string
}

func (f *fixture) git(args ...string) string {
	f.t.Helper()
	full := append([]string{"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = f.dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "HOME="+f.dir,
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit writes files (path → content) and commits them with msg.
func (f *fixture) commit(msg string, files map[string]string) string {
	f.t.Helper()
	for p, c := range files {
		full := filepath.Join(f.dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			f.t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	f.git("add", "-A")
	f.git("commit", "-q", "-m", msg)
	return f.git("rev-parse", "HEAD")
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t, dir: t.TempDir()}
	f.repo = gitx.Repo{Dir: f.dir}
	f.git("init", "-q")
	f.base = f.commit("init", map[string]string{
		"a/BUILD": "# a\n", "a/src.txt": "a\n", "a/bump.txt": "v=1\n", "a/notes.md": "n\n",
		"a/VERSION": "1.0.0\n", "a/CHANGELOG.md": "# a\n",
		"b/BUILD": "# b\n", "b/src.txt": "b\n", "b/VERSION": "1.0.0\n", "b/CHANGELOG.md": "# b\n",
		"lib/BUILD": "# lib\n", "lib/x": "x\n",
	})
	f.run = &fakeRunner{labels: map[string][]string{universe: {"@@//a:t", "@@//b:t"}}, nodes: deps}
	f.comps = []component.Component{
		{Label: "//a:a", Name: "a", Package: "a", Targets: []string{"//a:t"}, VersionFile: "a/VERSION", Changelog: "a/CHANGELOG.md"},
		{Label: "//b:b", Name: "b", Package: "b", Targets: []string{"//b:t"}, VersionFile: "b/VERSION", Changelog: "b/CHANGELOG.md"},
	}
	return f
}

func (f *fixture) states(base string) map[string]component.State {
	out := map[string]component.State{}
	for _, c := range f.comps {
		out[c.Label] = component.State{BaselineSHA: base}
	}
	return out
}

func (f *fixture) attribute(states map[string]component.State) (map[string]Result, []string) {
	f.t.Helper()
	head := f.git("rev-parse", "HEAD")
	res, warns, err := Attribute(cache.Options{Workspace: f.dir, Runner: f.run, Repo: f.repo}, f.repo, head, f.comps, states, nil)
	if err != nil {
		f.t.Fatalf("Attribute: %v", err)
	}
	return res, warns
}

// shas returns the attributed commit SHAs of label, newest first.
func shas(t *testing.T, res map[string]Result, label string) []string {
	t.Helper()
	r, ok := res[label]
	if !ok {
		t.Fatalf("no result for %s: %v", label, res)
	}
	var out []string
	for _, c := range r.Commits {
		out = append(out, c.SHA)
	}
	return out
}

func eq(t *testing.T, what string, got, want []string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// --- tests ---

func TestSharedDependencyReleasesBoth(t *testing.T) {
	f := newFixture(t)
	c1 := f.commit("feat: lib", map[string]string{"lib/x": "x2\n"})
	c2 := f.commit("fix: lib again", map[string]string{"lib/x": "x3\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), []string{c2, c1})
	eq(t, "b", shas(t, res, "//b:b"), []string{c2, c1})
	if got := res["//a:a"].Files[c1]; !reflect.DeepEqual(got, []string{"lib/x"}) {
		t.Errorf("files of %s = %v", c1, got)
	}
	if res["//a:a"].Component.Name != "a" {
		t.Errorf("component = %+v", res["//a:a"].Component)
	}
	eq(t, "a targets", res["//a:a"].Targets, []string{"//a:t"})
}

func TestOwnDirOnly(t *testing.T) {
	f := newFixture(t)
	c1 := f.commit("feat: a", map[string]string{"a/src.txt": "a2\n"})
	c2 := f.commit("feat: b", map[string]string{"b/src.txt": "b2\n", "a/src.txt": "a3\n"})
	f.commit("docs: unrelated", map[string]string{"README.md": "r\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), []string{c2, c1})
	eq(t, "b", shas(t, res, "//b:b"), []string{c2})
	if got := res["//b:b"].Files[c2]; !reflect.DeepEqual(got, []string{"b/src.txt"}) {
		t.Errorf("b files of %s = %v", c2, got)
	}
	if got := res["//a:a"].Files[c2]; !reflect.DeepEqual(got, []string{"a/src.txt"}) {
		t.Errorf("a files of %s = %v", c2, got)
	}
}

func TestReleaseCommitIgnored(t *testing.T) {
	f := newFixture(t)
	// Even touching a source file, a release commit never counts.
	f.commit("chore(release): a 1.1.0", map[string]string{"a/VERSION": "1.1.0\n", "a/CHANGELOG.md": "# a\n## 1.1.0\n", "lib/x": "x2\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), nil)
	eq(t, "b", shas(t, res, "//b:b"), nil)

	// A subject that merely looks similar is an ordinary commit.
	c := f.commit("chore(release): bump the lib", map[string]string{"lib/x": "x3\n"})
	res, _ = f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), []string{c})
}

func TestOtherComponentsVersionFileIgnored(t *testing.T) {
	f := newFixture(t)
	// b depends on a/VERSION through //a:version, but a version file of any
	// component is never attributed, even outside a release commit.
	f.commit("fix: hand edit", map[string]string{"a/VERSION": "1.0.1\n", "a/CHANGELOG.md": "edited\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), nil)
	eq(t, "b", shas(t, res, "//b:b"), nil)
}

func TestBumpFilesAttributedToOwner(t *testing.T) {
	f := newFixture(t)
	f.comps[0].BumpFiles = map[string]string{"a/bump.txt": `^v=(.*)$`}
	// A release commit editing the bump file never counts.
	f.commit("chore(release): a 1.1.0", map[string]string{"a/bump.txt": "v=1.1.0\n", "a/VERSION": "1.1.0\n"})
	// A non-release edit of the bump file is a real change of its owner.
	c := f.commit("build: bump dep", map[string]string{"a/bump.txt": "v=2\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), []string{c})
	eq(t, "a files", res["//a:a"].Files[c], []string{"a/bump.txt"})
}

func TestReleaseCommitSquashSuffixIgnored(t *testing.T) {
	f := newFixture(t)
	f.commit("chore(release): a 1.1.0 (#12)", map[string]string{"a/VERSION": "1.1.0\n", "lib/x": "x2\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), nil)
	eq(t, "b", shas(t, res, "//b:b"), nil)
}

func TestParseReleaseSubject(t *testing.T) {
	for _, tc := range []struct {
		subject, name, version string
		ok                     bool
	}{
		{"chore(release): a 1.1.0", "a", "1.1.0", true},
		{"chore(release): my-api 2.0.10 (#12)", "my-api", "2.0.10", true},
		{"chore(release): update workflow", "", "", false},
		{"chore(release): a 1.1", "", "", false},
		{"chore(release): a 01.1.0", "", "", false},
		{"chore(release): a 1.1.0-rc.1", "", "", false},
		{"chore(release): a 1.1.0 (#12) extra", "", "", false},
		{"fix: a 1.1.0", "", "", false},
	} {
		name, v, ok := ParseReleaseSubject(tc.subject)
		if ok != tc.ok || (ok && (name != tc.name || v.String() != tc.version)) {
			t.Errorf("ParseReleaseSubject(%q) = %q, %v, %v; want %q, %q, %v", tc.subject, name, v, ok, tc.name, tc.version, tc.ok)
		}
	}
}

func TestReleaseSubjectOfUnknownComponentNotIgnored(t *testing.T) {
	f := newFixture(t)
	// "prepare" is not a component: this is an ordinary commit.
	c := f.commit("chore(release): prepare 1.0.0", map[string]string{"a/src.txt": "a2\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), []string{c})
}

func TestInvalidReleaseVersionNotIgnored(t *testing.T) {
	f := newFixture(t)
	c := f.commit("chore(release): update workflow", map[string]string{"lib/x": "x2\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), []string{c})
}

func TestMergeCommitNotAttributed(t *testing.T) {
	f := newFixture(t)
	f.git("checkout", "-q", "-b", "side")
	c1 := f.commit("feat: side", map[string]string{"lib/x": "x2\n"})
	f.git("checkout", "-q", "main")
	c2 := f.commit("feat: main", map[string]string{"a/src.txt": "a2\n"})
	f.git("merge", "-q", "--no-ff", "-m", "Merge side", "side")
	merge := f.git("rev-parse", "HEAD")
	res, _ := f.attribute(f.states(f.base))
	got := shas(t, res, "//a:a")
	for _, s := range got {
		if s == merge {
			t.Errorf("merge commit attributed: %v", got)
		}
	}
	if len(got) != 2 || !(contains(got, c1) && contains(got, c2)) {
		t.Errorf("a = %v, want %s and %s", got, c1, c2)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func TestEmptyMembershipIsError(t *testing.T) {
	f := newFixture(t)
	f.comps[1].Targets = nil
	f.comps[1].TargetsQuery = "kind(rule, //b:none*)"
	f.run.labels = map[string][]string{
		`set("//a:t") + (kind(rule, //b:none*))`: {"@@//a:t"},
		`(kind(rule, //b:none*))`:                {},
	}
	_, _, err := Attribute(cache.Options{Workspace: f.dir, Runner: f.run, Repo: f.repo}, f.repo, f.base, f.comps, f.states(f.base), nil)
	if err == nil || !strings.Contains(err.Error(), "//b:b") || !strings.Contains(err.Error(), "kind(rule, //b:none*)") {
		t.Fatalf("err = %v", err)
	}

	f = newFixture(t)
	f.comps[1].Targets = nil
	f.run.labels = map[string][]string{`set("//a:t") + set()`: {"@@//a:t"}}
	_, _, err = Attribute(cache.Options{Workspace: f.dir, Runner: f.run, Repo: f.repo}, f.repo, f.base, f.comps, f.states(f.base), nil)
	if err == nil || !strings.Contains(err.Error(), "//b:b") {
		t.Fatalf("err = %v", err)
	}
}

func TestCallerWarnAlsoCalled(t *testing.T) {
	f := newFixture(t)
	states := f.states(f.base)
	states["//a:a"] = component.State{BaselineSHA: "0123456789012345678901234567890123456789"}
	var got []string
	o := cache.Options{Workspace: f.dir, Runner: f.run, Repo: f.repo, Warn: func(s string) { got = append(got, s) }}
	_, warns, err := Attribute(o, f.repo, f.base, f.comps, states, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) == 0 || !reflect.DeepEqual(got, warns) {
		t.Errorf("caller warn got %v, returned %v", got, warns)
	}
}

func TestIgnoreGlobs(t *testing.T) {
	f := newFixture(t)
	f.comps[0].Ignore = []string{"a/**/*.md"}
	f.commit("docs: notes", map[string]string{"a/notes.md": "n2\n"})
	c := f.commit("feat: src and notes", map[string]string{"a/notes.md": "n3\n", "a/src.txt": "a2\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), []string{c})
	eq(t, "a files", res["//a:a"].Files[c], []string{"a/src.txt"})
}

func TestRootBaselineForNewComponent(t *testing.T) {
	f := newFixture(t)
	c1 := f.commit("feat: lib", map[string]string{"lib/x": "x2\n"})
	tagged := f.git("rev-parse", "HEAD")
	c2 := f.commit("feat: b only", map[string]string{"b/src.txt": "b2\n"})
	states := map[string]component.State{
		"//a:a": {BaselineSHA: report.RootBaseline},
		"//b:b": {BaselineSHA: tagged},
	}
	res, _ := f.attribute(states)
	// The root commit also touches lib/x and a/src.txt.
	eq(t, "a", shas(t, res, "//a:a"), []string{c1, f.base})
	eq(t, "b", shas(t, res, "//b:b"), []string{c2})
}

func TestTargetsQuery(t *testing.T) {
	f := newFixture(t)
	f.comps[1].Targets = nil
	f.comps[1].TargetsQuery = "kind(rule, //b:*)"
	f.run.labels = map[string][]string{
		`set("//a:t") + (kind(rule, //b:*))`: {"@@//a:t", "@@//b:t"},
		`(kind(rule, //b:*))`:                {"@@//b:t"},
	}
	c := f.commit("feat: b", map[string]string{"b/src.txt": "b2\n"})
	res, _ := f.attribute(f.states(f.base))
	eq(t, "a", shas(t, res, "//a:a"), nil)
	eq(t, "b", shas(t, res, "//b:b"), []string{c})
	eq(t, "b targets", res["//b:b"].Targets, []string{"//b:t"})
	n := 0
	for _, e := range f.run.calls {
		if e == `(kind(rule, //b:*))` {
			n++
		}
	}
	if n != 1 {
		t.Errorf("targets_query ran %d times (calls %v)", n, f.run.calls)
	}
}

func TestTargetOutsideUniverseIsError(t *testing.T) {
	f := newFixture(t)
	f.comps[0].Targets = []string{"//a:t", "//a:gone"}
	f.run.labels = map[string][]string{`set("//a:t" "//a:gone") + set("//b:t")`: {"@@//a:t", "@@//b:t"}}
	_, _, err := Attribute(cache.Options{Workspace: f.dir, Runner: f.run, Repo: f.repo}, f.repo, f.base, f.comps, f.states(f.base), nil)
	if err == nil || !strings.Contains(err.Error(), "//a:gone") {
		t.Fatalf("err = %v, want one naming //a:gone", err)
	}
}

func TestMissingStateIsError(t *testing.T) {
	f := newFixture(t)
	states := f.states(f.base)
	delete(states, "//b:b")
	_, _, err := Attribute(cache.Options{Workspace: f.dir, Runner: f.run, Repo: f.repo}, f.repo, f.base, f.comps, states, nil)
	if err == nil || !strings.Contains(err.Error(), "//b:b") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnresolvableBaselineFallsBackToRoot(t *testing.T) {
	f := newFixture(t)
	c := f.commit("feat: a", map[string]string{"a/src.txt": "a2\n"})
	states := f.states(f.base)
	states["//a:a"] = component.State{BaselineSHA: "0123456789012345678901234567890123456789"}
	res, warns := f.attribute(states)
	eq(t, "a", shas(t, res, "//a:a"), []string{c, f.base})
	found := false
	for _, w := range warns {
		if strings.Contains(w, "//a:a") && strings.Contains(w, "all history") {
			found = true
		}
	}
	if !found {
		t.Errorf("no fallback warning in %v", warns)
	}
}

func TestGlobalGlobs(t *testing.T) {
	f := newFixture(t)
	c := f.commit("build: tooling", map[string]string{"tools/x.sh": "x\n"})
	head := f.git("rev-parse", "HEAD")
	res, _, err := Attribute(cache.Options{Workspace: f.dir, Runner: f.run, Repo: f.repo}, f.repo, head, f.comps, f.states(f.base), []string{"tools/**"})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "a", shas(t, res, "//a:a"), []string{c})
	eq(t, "b", shas(t, res, "//b:b"), []string{c})
}

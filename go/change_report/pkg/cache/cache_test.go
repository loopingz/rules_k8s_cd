package cache

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
)

// gitCmd runs a git command in dir, isolated from user/system git config.
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

type repo struct{ gitx.Repo }

func newRepo(t *testing.T) repo {
	t.Helper()
	dir := t.TempDir()
	gitCmd(t, dir, "init", "-q")
	return repo{gitx.Repo{Dir: dir}}
}

func (r repo) write(t *testing.T, path, content string) {
	t.Helper()
	p := filepath.Join(r.Dir, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (r repo) commit(t *testing.T, msg string) {
	t.Helper()
	gitCmd(t, r.Dir, "add", "-A")
	gitCmd(t, r.Dir, "commit", "-q", "-m", msg)
}

// fakeRunner serves canned `bazel` output and records every call.
type fakeRunner struct {
	labels []string
	nodes  []string // streamed_jsonproto lines
	// partial, if set, serves `deps(set(...))` queries instead of nodes.
	partial func(expr string) []string
	fail    func(args []string) bool
	calls   [][]string
}

func (f *fakeRunner) Run(args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if f.fail != nil && f.fail(args) {
		return nil, errors.New("fake failure")
	}
	switch {
	case len(args) > 0 && args[0] == "version":
		return []byte("Build label: 8.0.0\n"), nil
	case contains(args, "--output=label"):
		return []byte(strings.Join(f.labels, "\n") + "\n"), nil
	case contains(args, "--output=streamed_jsonproto") && f.partial != nil && strings.HasPrefix(args[len(args)-1], "deps(set("):
		return []byte(strings.Join(f.partial(args[len(args)-1]), "\n") + "\n"), nil
	case contains(args, "--output=streamed_jsonproto"):
		return []byte(strings.Join(f.nodes, "\n") + "\n"), nil
	}
	return nil, fmt.Errorf("fake: unexpected call %v", args)
}

func (f *fakeRunner) queryExprs() []string {
	var out []string
	for _, c := range f.calls {
		if len(c) > 0 && c[0] == "query" {
			out = append(out, c[len(c)-1])
		}
	}
	return out
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func src(l string) string {
	return fmt.Sprintf(`{"type":"SOURCE_FILE","sourceFile":{"name":%q}}`, l)
}

func rule(l string, inputs ...string) string {
	q := make([]string, len(inputs))
	for i, in := range inputs {
		q[i] = fmt.Sprintf("%q", in)
	}
	return fmt.Sprintf(`{"type":"RULE","rule":{"name":%q,"ruleClass":"filegroup","ruleInput":[%s]}}`, l, strings.Join(q, ","))
}

type fixture struct {
	r      repo
	run    *fakeRunner
	opts   Options
	warns  []string
	cacheD string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	r := newRepo(t)
	r.write(t, "MODULE.bazel", "module(name = \"self\")\n")
	r.write(t, "MODULE.bazel.lock", "{}\n")
	r.write(t, "pkg/BUILD", "load(\":m.bzl\", \"m\")\nload(\"@self//lib:n.bzl\", \"n\")\nm(name = \"viamacro\")\n")
	r.write(t, "pkg/m.bzl", "def m(name): pass\n")
	r.write(t, "lib/n.bzl", "n = 1\n")
	r.write(t, "pkg/manifests/a.yaml", "a: 1\n")
	r.write(t, "pkg/manifests/b.yaml", "b: 1\n")
	r.write(t, "sub/BUILD", "genrule(name = \"g\")\n")
	r.write(t, "sub/x.txt", "x\n")
	r.write(t, "README.md", "no package\n")
	r.commit(t, "init")
	run := &fakeRunner{
		labels: []string{"@@//pkg:viamacro", "@@//sub:g"},
		nodes: []string{
			src("@@//pkg:manifests/a.yaml"),
			src("@@//pkg:manifests/b.yaml"),
			rule("@@//pkg:viamacro", "@@//pkg:manifests/a.yaml", "@@//pkg:manifests/b.yaml"),
			rule("@@//sub:g", "@@//sub:x.txt", "@@bazel_tools//tools/genrule:genrule-setup.sh"),
			src("@@//sub:x.txt"),
		},
	}
	f := &fixture{r: r, run: run, cacheD: filepath.Join(t.TempDir(), "cache")}
	f.opts = Options{
		Workspace: r.Dir,
		CacheDir:  f.cacheD,
		Universe:  "//...",
		Runner:    run,
		Repo:      r.Repo,
		Warn:      func(s string) { f.warns = append(f.warns, s) },
	}
	return f
}

func (f *fixture) load(t *testing.T, wantMode string) (*Snapshot, Stats) {
	t.Helper()
	f.run.calls = nil
	s, st, err := Load(f.opts)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Mode != wantMode {
		t.Fatalf("mode = %q, want %q (calls %v, warns %v)", st.Mode, wantMode, f.run.calls, f.warns)
	}
	return s, st
}

func TestFullThenHit(t *testing.T) {
	f := setup(t)
	s, _ := f.load(t, "full")
	if !reflect.DeepEqual(s.Universe, []string{"//pkg:viamacro", "//sub:g"}) {
		t.Errorf("universe = %v", s.Universe)
	}
	c, ok := s.Closures["//pkg:viamacro"]
	if !ok {
		t.Fatalf("no closure for //pkg:viamacro: %v", s.Closures)
	}
	wantFiles := []string{"lib/n.bzl", "pkg/BUILD", "pkg/m.bzl", "pkg/manifests/a.yaml", "pkg/manifests/b.yaml"}
	if !reflect.DeepEqual(c.Files, wantFiles) {
		t.Errorf("closure files = %v, want %v", c.Files, wantFiles)
	}
	if g := s.Closures["//sub:g"]; !reflect.DeepEqual(g.Repos, []string{"bazel_tools"}) {
		t.Errorf("sub:g repos = %v", g.Repos)
	}
	if s.Tree["pkg/BUILD"] == "" || s.GlobalKey == "" || len(s.PackageKeys) != 2 {
		t.Errorf("snapshot incomplete: tree=%d key=%q pkgkeys=%v", len(s.Tree), s.GlobalKey, s.PackageKeys)
	}
	if s.Mapping.FileRepos == nil {
		t.Error("mapping not loaded")
	}

	s2, _ := f.load(t, "hit")
	if len(f.run.calls) != 0 {
		t.Errorf("hit made runner calls: %v", f.run.calls)
	}
	if !reflect.DeepEqual(s2.Closures, s.Closures) || !reflect.DeepEqual(s2.Universe, s.Universe) {
		t.Error("hit snapshot differs from full snapshot")
	}
	if s2.Tree["pkg/BUILD"] == "" {
		t.Error("hit snapshot has no tree")
	}
	if !reflect.DeepEqual(s2.Graph.Loads["pkg/BUILD"], []string{"pkg/m.bzl", "lib/n.bzl"}) {
		t.Errorf("persisted loads = %v", s2.Graph.Loads)
	}
}

func TestEditDoesNotInvalidate(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	f.r.write(t, "pkg/manifests/a.yaml", "a: 2\n")
	f.r.commit(t, "edit")
	f.load(t, "hit")
}

func TestAddFileRequeriesOnlyPackage(t *testing.T) {
	f := setup(t)
	s1, _ := f.load(t, "full")
	f.r.write(t, "pkg/manifests/c.yaml", "c: 1\n")
	f.r.commit(t, "add c")
	f.run.nodes = []string{
		src("@@//pkg:manifests/a.yaml"),
		src("@@//pkg:manifests/b.yaml"),
		src("@@//pkg:manifests/c.yaml"),
		rule("@@//pkg:viamacro", "@@//pkg:manifests/a.yaml", "@@//pkg:manifests/b.yaml", "@@//pkg:manifests/c.yaml"),
	}
	s, st := f.load(t, "partial")
	if !reflect.DeepEqual(st.RequeriedPackages, []string{"pkg"}) {
		t.Errorf("requeried = %v", st.RequeriedPackages)
	}
	exprs := f.run.queryExprs()
	var deps string
	for _, e := range exprs {
		if strings.HasPrefix(e, "deps(") {
			deps = e
		}
	}
	if deps != "deps(set(//pkg:*))" {
		t.Errorf("deps query = %q (all: %v)", deps, exprs)
	}
	if !contains(s.Closures["//pkg:viamacro"].Files, "pkg/manifests/c.yaml") {
		t.Errorf("closure not recomputed: %v", s.Closures["//pkg:viamacro"].Files)
	}
	if !reflect.DeepEqual(s.Closures["//sub:g"], s1.Closures["//sub:g"]) {
		t.Errorf("unrelated closure changed: %v", s.Closures["//sub:g"])
	}
	if _, ok := s.Graph.Nodes["@@//sub:g"]; !ok {
		t.Error("unchanged package nodes dropped")
	}
	// The partial result is persisted: next run is a hit.
	f.load(t, "hit")
}

func TestBuildEditRecomputesLoads(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	f.r.write(t, "pkg/BUILD", "load(\":m.bzl\", \"m\")\nm(name = \"viamacro\")\n")
	f.r.commit(t, "drop n.bzl load")
	s, _ := f.load(t, "partial")
	if !reflect.DeepEqual(s.Graph.Loads["pkg/BUILD"], []string{"pkg/m.bzl"}) {
		t.Errorf("loads = %v", s.Graph.Loads["pkg/BUILD"])
	}
	if contains(s.Closures["//pkg:viamacro"].Files, "lib/n.bzl") {
		t.Errorf("closure still has dropped load: %v", s.Closures["//pkg:viamacro"].Files)
	}
}

func TestPartialFailureFallsBackToFull(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	f.r.write(t, "pkg/manifests/c.yaml", "c: 1\n")
	f.r.commit(t, "add c")
	f.run.fail = func(args []string) bool { return strings.HasPrefix(args[len(args)-1], "deps(set(") }
	f.load(t, "full")
}

func TestUniverseChangeFull(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	f.r.write(t, "pkg/BUILD", "load(\":m.bzl\", \"m\")\nm(name = \"viamacro\")\nm(name = \"other\")\n")
	f.r.commit(t, "new target")
	f.run.labels = append(f.run.labels, "@@//pkg:other")
	s, _ := f.load(t, "full")
	if _, ok := s.Closures["//pkg:other"]; !ok {
		t.Error("new target has no closure")
	}
}

func TestNewPackageFull(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	f.r.write(t, "new/BUILD", "")
	f.r.commit(t, "new pkg")
	f.load(t, "full")
}

func TestRemovedPackageFull(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	gitCmd(t, f.r.Dir, "rm", "-q", "sub/BUILD")
	f.r.commit(t, "rm pkg")
	f.load(t, "full")
}

func TestBzlChangeFull(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	f.r.write(t, "pkg/m.bzl", "def m(name): print(1)\n")
	f.r.commit(t, "edit bzl")
	f.load(t, "full")
}

func gen(l, rule string) string {
	return fmt.Sprintf(`{"type":"GENERATED_FILE","generatedFile":{"name":%q,"generatingRule":%q}}`, l, rule)
}

func TestPartialKeepsCrossPackageGeneratedFiles(t *testing.T) {
	f := setup(t)
	f.r.write(t, "p/BUILD", "genrule(name = \"genrule\")\n")
	f.r.write(t, "p/in.txt", "in\n")
	f.r.write(t, "q/BUILD", "filegroup(name = \"t\")\n")
	f.r.commit(t, "gen")
	pNodes := []string{
		src("@@//p:in.txt"),
		rule("@@//p:genrule", "@@//p:in.txt"),
		gen("@@//p:gen.out", "@@//p:genrule"),
	}
	f.run.labels = append(f.run.labels, "@@//p:genrule", "@@//q:t")
	f.run.nodes = append(append(f.run.nodes, pNodes...), rule("@@//q:t", "@@//p:gen.out"))
	// Like Bazel: `//p:all` matches rules only, `//p:*` every target.
	f.run.partial = func(expr string) []string {
		if strings.Contains(expr, "//p:*") {
			return pNodes
		}
		return pNodes[1:2]
	}
	f.load(t, "full")
	f.r.write(t, "p/extra.txt", "x\n")
	f.r.commit(t, "add file to p")
	s, st := f.load(t, "partial")
	if !reflect.DeepEqual(st.RequeriedPackages, []string{"p"}) {
		t.Errorf("requeried = %v", st.RequeriedPackages)
	}
	if c := s.Closures["//q:t"]; !contains(c.Files, "p/in.txt") {
		t.Errorf("//q:t closure lost p/in.txt: %v", c.Files)
	}
}

func TestUntrackedLockfileChangeFull(t *testing.T) {
	f := setup(t)
	gitCmd(t, f.r.Dir, "rm", "-q", "--cached", "MODULE.bazel.lock")
	gitCmd(t, f.r.Dir, "commit", "-q", "-m", "untrack lockfile")
	if _, tracked := mustTree(t, f.r)["MODULE.bazel.lock"]; tracked {
		t.Fatal("lockfile still tracked")
	}
	f.load(t, "full")
	f.load(t, "hit")
	f.r.write(t, "MODULE.bazel.lock", `{"lockFileVersion": 1}`+"\n")
	f.load(t, "full")
	f.load(t, "hit")
}

func mustTree(t *testing.T, r repo) map[string]string {
	t.Helper()
	tree, err := r.TreeFiles("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestGoModEditFull(t *testing.T) {
	f := setup(t)
	f.r.write(t, "sub/go.mod", "module x\n")
	f.r.commit(t, "go.mod")
	f.load(t, "full")
	f.r.write(t, "sub/go.mod", "module x\n\nrequire y v1.0.0\n")
	f.r.commit(t, "bump")
	f.load(t, "full")
}

func TestMappedFileEditFull(t *testing.T) {
	f := setup(t)
	f.r.write(t, "MODULE.bazel.lock", `{"moduleExtensions":{"//ext:e.bzl%e":{"general":{"recordedInputs":["FILE:@@//sub/images.yaml abc"],"generatedRepoSpecs":{"img":{}}}}}}`+"\n")
	f.r.write(t, "sub/images.yaml", "a: 1\n")
	f.r.commit(t, "mapped file")
	s, _ := f.load(t, "full")
	if _, ok := s.Mapping.FileRepos["sub/images.yaml"]; !ok {
		t.Fatalf("images.yaml not mapped: %v", s.Mapping.FileRepos)
	}
	f.load(t, "hit")
	f.r.write(t, "sub/images.yaml", "a: 2\n")
	f.r.commit(t, "edit images")
	f.load(t, "full")
}

func TestBazelignoreChangeFull(t *testing.T) {
	f := setup(t)
	f.r.write(t, ".bazelignore", "a\n")
	f.r.commit(t, "ignore")
	f.load(t, "full")
	f.r.write(t, ".bazelignore", "a\nb\n")
	f.r.commit(t, "ignore more")
	f.load(t, "full")
}

func TestUniverseExprChangeFull(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	f.opts.Universe = "//pkg/..."
	f.load(t, "full")
}

func TestCorruptCache(t *testing.T) {
	f := setup(t)
	f.load(t, "full")
	ents, err := os.ReadDir(f.cacheD)
	if err != nil || len(ents) != 1 || !strings.HasSuffix(ents[0].Name(), ".json") {
		t.Fatalf("cache dir: %v %v", ents, err)
	}
	if len(strings.TrimSuffix(ents[0].Name(), ".json")) != 16 {
		t.Errorf("cache file name %q", ents[0].Name())
	}
	p := filepath.Join(f.cacheD, ents[0].Name())
	if err := os.WriteFile(p, []byte(`{"FormatVer`), 0o644); err != nil {
		t.Fatal(err)
	}
	f.load(t, "full")
	f.load(t, "hit")
}

func TestUnwritableCacheDir(t *testing.T) {
	f := setup(t)
	if err := os.WriteFile(f.cacheD, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.load(t, "full")
	saved := false
	for _, w := range f.warns {
		saved = saved || strings.HasPrefix(w, "saving graph cache")
	}
	if !saved {
		t.Errorf("warnings = %q, want a save failure", f.warns)
	}
}

func TestFullFailureIsError(t *testing.T) {
	f := setup(t)
	f.run.fail = func(args []string) bool { return args[0] == "query" }
	if _, _, err := Load(f.opts); err == nil {
		t.Error("expected error")
	}
}

func TestPackagesAndKeys(t *testing.T) {
	tree := map[string]string{
		"BUILD.bazel":      "r",
		"top.txt":          "t",
		"a/BUILD":          "a1",
		"a/x/y.txt":        "y",
		"a/b/BUILD.bazel":  "b1",
		"a/b/c.txt":        "c",
		"nopkg/z":          "z",
		"a/m.bzl":          "m",
		"MODULE.bazel":     "mod",
		".bazelrc":         "rc",
		"WORKSPACE.bzlmod": "ws",
	}
	pkgs := Packages(tree)
	if !reflect.DeepEqual(pkgs, map[string]bool{"": true, "a": true, "a/b": true}) {
		t.Fatalf("packages = %v", pkgs)
	}
	ka := PackageKey("a", tree, pkgs)
	// Content edits do not change the key; adding a file does; editing BUILD does.
	tree["a/x/y.txt"] = "y2"
	if PackageKey("a", tree, pkgs) != ka {
		t.Error("content edit changed key")
	}
	tree["a/x/new.txt"] = "n"
	if PackageKey("a", tree, pkgs) == ka {
		t.Error("added file did not change key")
	}
	kb := PackageKey("a/b", tree, pkgs)
	tree["a/b/d.txt"] = "d"
	if PackageKey("a", tree, pkgs) == ka || PackageKey("a/b", tree, pkgs) == kb {
		t.Error("sub-package file attribution wrong")
	}
	ka = PackageKey("a", tree, pkgs)
	tree["a/b/e.txt"] = "e"
	if PackageKey("a", tree, pkgs) != ka {
		t.Error("file in sub-package changed parent key")
	}
	tree["a/BUILD"] = "a2"
	if PackageKey("a", tree, pkgs) == ka {
		t.Error("BUILD edit did not change key")
	}
	kr := PackageKey("", tree, pkgs)
	tree["nopkg/zz"] = "zz"
	if PackageKey("", tree, pkgs) == kr {
		t.Error("root package does not own nopkg/")
	}

	g := GlobalKey("//...", "8.0.0", tree)
	if GlobalKey("//...", "8.0.0", tree) != g {
		t.Error("global key not deterministic")
	}
	for _, mut := range []struct{ k, v string }{
		{"a/m.bzl", "m2"}, {"MODULE.bazel", "mod2"}, {".bazelrc", "rc2"}, {"WORKSPACE.bzlmod", "ws2"}, {"MODULE.bazel.lock", "l"}, {"x/new.bzl", "n"},
		{".bazelignore", "i"}, {"REPO.bazel", "r"}, {"tools/ci.bazelrc", "c"}, {"deps/go.MODULE.bazel", "g"},
	} {
		old, had := tree[mut.k]
		tree[mut.k] = mut.v
		if GlobalKey("//...", "8.0.0", tree) == g {
			t.Errorf("changing %s did not change global key", mut.k)
		}
		if had {
			tree[mut.k] = old
		} else {
			delete(tree, mut.k)
		}
	}
	tree["a/x/y.txt"] = "y3"
	if GlobalKey("//...", "8.0.0", tree) != g {
		t.Error("non-global file changed global key")
	}
	if GlobalKey("//pkg/...", "8.0.0", tree) == g || GlobalKey("//...", "8.1.0", tree) == g {
		t.Error("universe/version not in global key")
	}
}

func TestRepoNameIsMainRepo(t *testing.T) {
	f := setup(t)
	f.r.write(t, "MODULE.bazel", "module(\n    name = \"self\",\n    repo_name = \"y\",\n)\n")
	f.r.write(t, "pkg/BUILD", "load(\":m.bzl\", \"m\")\nload(\"@y//lib:n.bzl\", \"n\")\nm(name = \"viamacro\")\n")
	f.r.commit(t, "repo_name")
	s, _ := f.load(t, "full")
	if !contains(s.Closures["//pkg:viamacro"].Files, "lib/n.bzl") {
		t.Errorf("closure files = %v, want lib/n.bzl loaded through repo_name", s.Closures["//pkg:viamacro"].Files)
	}
}

func TestBzlInSubdirLoadsRelativeToPackage(t *testing.T) {
	f := setup(t)
	f.r.write(t, "pkg/defs/x.bzl", "load(\":m.bzl\", \"m\")\n")
	f.r.commit(t, "bzl in subdir")
	s, _ := f.load(t, "full")
	if got := s.Graph.Loads["pkg/defs/x.bzl"]; !reflect.DeepEqual(got, []string{"pkg/m.bzl"}) {
		t.Errorf("loads of pkg/defs/x.bzl = %v, want [pkg/m.bzl]", got)
	}
}

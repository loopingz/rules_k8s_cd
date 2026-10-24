package extdeps

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func read(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFromLockfile(t *testing.T) {
	m, err := FromLockfile(read(t, "testdata/MODULE.bazel.lock"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gazelle++go_deps+com_github_foo_bar", "gazelle++go_deps+org_golang_x_text"}
	for _, f := range []string{"go.mod", "go.sum"} {
		if !reflect.DeepEqual(m.FileRepos[f], want) {
			t.Errorf("%s: got %v want %v", f, m.FileRepos[f], want)
		}
	}
	if len(m.FileRepos) != 2 {
		t.Errorf("unexpected files: %v", m.FileRepos)
	}
}

func TestFromLockfileRootPrefix(t *testing.T) {
	lock := `{"moduleExtensions":{"//lib:extensions.bzl%toolchains":{"general":{"recordedInputs":["FILE:@@//lib/x.txt abc"],"generatedRepoSpecs":{"helm":{}}}}}}`
	m, err := FromLockfile([]byte(lock))
	if err != nil {
		t.Fatal(err)
	}
	if got := m.FileRepos["lib/x.txt"]; !reflect.DeepEqual(got, []string{"+toolchains+helm"}) {
		t.Errorf("got %v", got)
	}
}

func TestFromLockfileBad(t *testing.T) {
	if _, err := FromLockfile([]byte("{nope")); err == nil {
		t.Error("expected error")
	}
}

func TestCanonicalPrefix(t *testing.T) {
	cases := map[string]string{
		"//lib:extensions.bzl%toolchains":                      "+toolchains+",
		"@@//lib:extensions.bzl%toolchains":                    "+toolchains+",
		"@@rules_oci+//oci:extensions.bzl%oci":                 "rules_oci++oci+",
		"@@aspect_bazel_lib+//lib:extensions.bzl%toolchains":   "aspect_bazel_lib++toolchains+",
		"@@rules_python+//python/extensions:config.bzl%config": "rules_python++config+",
	}
	for id, want := range cases {
		got, ok := canonicalPrefix(id)
		if !ok || got != want {
			t.Errorf("%s: got %q,%v want %q", id, got, ok, want)
		}
	}
	if _, ok := canonicalPrefix("@rules_oci//oci:extensions.bzl%oci"); ok {
		t.Error("apparent id should not parse")
	}
}

func TestFromShowExtension(t *testing.T) {
	m := FromShowExtension(read(t, "testdata/show_extension.txt"))
	repos := []string{
		"aspect_rules_js++npm+npm",
		"aspect_rules_js++npm+npm__acorn__8.0.0",
		"aspect_rules_js++npm+npm__left-pad__1.3.0",
	}
	for _, f := range []string{"pnpm-lock.yaml", "package.json", "package-lock.json", "yarn.lock", "sub/package.json", "sub/pnpm-lock.yaml"} {
		if !reflect.DeepEqual(m.FileRepos[f], repos) {
			t.Errorf("%s: got %v", f, m.FileRepos[f])
		}
	}
	if _, ok := m.FileRepos["internal/file.txt"]; ok {
		t.Error("non-root usage must be ignored")
	}
	if _, ok := m.FileRepos[".bazelignore"]; !ok {
		t.Error("any root label attr is a file")
	}
}

func TestFromShowExtensionGoCompanions(t *testing.T) {
	out := "## @@gazelle+//internal/bzlmod:go_deps.bzl%go_deps:\n\nFetched repositories:\n  - a\n\n## Usage in <root> from /x/MODULE.bazel:3\ngo_deps.from_file(go_mod=\"//svc:go.mod\")\nuse_repo(\n  go_deps,\n  \"a\",\n)\n"
	m := FromShowExtension([]byte(out))
	for _, f := range []string{"svc/go.mod", "svc/go.sum"} {
		if !reflect.DeepEqual(m.FileRepos[f], []string{"gazelle++go_deps+a"}) {
			t.Errorf("%s: %v", f, m.FileRepos[f])
		}
	}
}

func TestRootExtensions(t *testing.T) {
	mb := `
module(name = "x")
# oci_x = use_extension("@commented//a.bzl", "no")
oci = use_extension("@rules_oci//oci:extensions.bzl", "oci")
toolchains = use_extension("//lib:extensions.bzl", "toolchains")
t2 = use_extension("//lib:extensions.bzl", "toolchains", dev_dependency = True)
`
	want := []string{"@rules_oci//oci:extensions.bzl%oci", "@@//lib:extensions.bzl%toolchains"}
	if got := RootExtensions([]byte(mb)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestRootExtensionsRootForms(t *testing.T) {
	mb := `module(name = "me", version = "1")
a = use_extension("//lib:x.bzl", "a")
b = use_extension("@//lib:x.bzl", "b")
c = use_extension("@me//lib:x.bzl", "c")
d = use_extension("@@//lib:x.bzl", "d")
e = use_extension("@other//lib:x.bzl", "e")
f = use_extension("@mealso//lib:x.bzl", "f")
`
	want := []string{"@@//lib:x.bzl%a", "@@//lib:x.bzl%b", "@@//lib:x.bzl%c", "@@//lib:x.bzl%d", "@other//lib:x.bzl%e", "@mealso//lib:x.bzl%f"}
	if got := RootExtensions([]byte(mb)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestLocalPaths(t *testing.T) {
	mb := `
local_path_override(
    module_name = "a",
    path = "../a",
)
local_repository(name = "b", path = "third_party/b")
git_override(module_name = "c", remote = "x", path = "nope")
`
	want := []string{"../a", "third_party/b"}
	if got := LocalPaths([]byte(mb)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestModuleRepos(t *testing.T) {
	mb := `bazel_dep(name = "rules_oci", version = "2.0")
bazel_dep(name="zlib", version="1", dev_dependency=True)
`
	if got := ModuleRepos([]byte(mb)); !reflect.DeepEqual(got, []string{"rules_oci+", "zlib+"}) {
		t.Errorf("got %v", got)
	}
}

func TestMerge(t *testing.T) {
	a := Mapping{FileRepos: map[string][]string{"go.sum": {"b", "a"}}, ModuleRepos: []string{"x+"}}
	b := Mapping{FileRepos: map[string][]string{"go.sum": {"c", "a"}, "p": {"z"}}, LocalPaths: []string{"l"}}
	got := a.Merge(b)
	want := Mapping{FileRepos: map[string][]string{"go.sum": {"a", "b", "c"}, "p": {"z"}}, ModuleRepos: []string{"x+"}, LocalPaths: []string{"l"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestChangedRepos(t *testing.T) {
	before := read(t, "testdata/MODULE.bazel.lock")
	after := []byte(strings.Replace(string(before), `"version": "v1.2.3"`, `"version": "v1.2.4"`, 1))
	got, ok := ChangedRepos(before, after)
	if !ok || !reflect.DeepEqual(got, []string{"gazelle++go_deps+com_github_foo_bar"}) {
		t.Errorf("got %v ok=%v", got, ok)
	}
	got, ok = ChangedRepos(before, before)
	if !ok || len(got) != 0 {
		t.Errorf("identical: got %v ok=%v", got, ok)
	}
}

func TestChangedReposAddedRemoved(t *testing.T) {
	before := []byte(`{"moduleExtensions":{"//:e.bzl%e":{"general":{"generatedRepoSpecs":{"a":{"x":1},"b":{"x":1}}}}}}`)
	after := []byte(`{"moduleExtensions":{"//:e.bzl%e":{"general":{"generatedRepoSpecs":{"b":{"x":1},"c":{"x":1}}}}}}`)
	got, ok := ChangedRepos(before, after)
	if !ok || !reflect.DeepEqual(got, []string{"+e+a", "+e+c"}) {
		t.Errorf("got %v ok=%v", got, ok)
	}
}

func TestChangedReposDigestOnly(t *testing.T) {
	before := []byte(`{"moduleExtensions":{"//:e.bzl%e":{"general":{"bzlTransitiveDigest":"1","generatedRepoSpecs":{"a":{},"b":{}}}}}}`)
	after := []byte(`{"moduleExtensions":{"//:e.bzl%e":{"general":{"bzlTransitiveDigest":"2","generatedRepoSpecs":{"a":{},"b":{}}}}}}`)
	got, ok := ChangedRepos(before, after)
	if !ok || !reflect.DeepEqual(got, []string{"+e+a", "+e+b"}) {
		t.Errorf("got %v ok=%v", got, ok)
	}
}

func TestChangedReposRegistry(t *testing.T) {
	before := []byte(`{"registryFileHashes":{"https://bcr.bazel.build/bazel_registry.json":"1","https://bcr.bazel.build/modules/zlib/1.3/MODULE.bazel":"a"}}`)
	after := []byte(`{"registryFileHashes":{"https://bcr.bazel.build/bazel_registry.json":"1","https://bcr.bazel.build/modules/zlib/1.3/MODULE.bazel":"b","https://bcr.bazel.build/modules/new/2/MODULE.bazel":"c"}}`)
	got, ok := ChangedRepos(before, after)
	if !ok || !reflect.DeepEqual(got, []string{"new+", "new++*", "new+2", "zlib+", "zlib++*", "zlib+1.3"}) {
		t.Errorf("got %v ok=%v", got, ok)
	}
	// bazel_registry.json itself changing cannot be attributed.
	after2 := []byte(strings.Replace(string(before), `registry.json":"1"`, `registry.json":"2"`, 1))
	if _, ok := ChangedRepos(before, after2); ok {
		t.Error("expected ok=false")
	}
	// unknown top-level difference
	if _, ok := ChangedRepos([]byte(`{"lockFileVersion":1}`), []byte(`{"lockFileVersion":2}`)); ok {
		t.Error("expected ok=false")
	}
}

func TestChangedReposInvalid(t *testing.T) {
	good := read(t, "testdata/MODULE.bazel.lock")
	if _, ok := ChangedRepos(good, []byte("{bad")); ok {
		t.Error("expected ok=false")
	}
	if _, ok := ChangedRepos([]byte("{bad"), good); ok {
		t.Error("expected ok=false")
	}
}

type fakeRunner struct {
	calls [][]string
	fn    func(args []string) ([]byte, error)
}

func (f *fakeRunner) Run(args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	return f.fn(args)
}

func TestLoad(t *testing.T) {
	ws := t.TempDir()
	show := read(t, "testdata/show_extension.txt")
	f := &fakeRunner{fn: func(a []string) ([]byte, error) { return show, nil }}
	mb := []byte(`bazel_dep(name = "rules_oci", version = "1")
npm = use_extension("@aspect_rules_js//npm:extensions.bzl", "npm")
local_path_override(module_name = "a", path = "../a")
`)
	m, warns, err := Load(ws, f, read(t, "testdata/MODULE.bazel.lock"), mb)
	if err != nil || len(warns) != 0 {
		t.Fatalf("err=%v warns=%v", err, warns)
	}
	if len(f.calls) != 1 || !reflect.DeepEqual(f.calls[0], []string{"mod", "show_extension", "@aspect_rules_js//npm:extensions.bzl%npm"}) {
		t.Errorf("calls=%v", f.calls)
	}
	if len(m.FileRepos["go.sum"]) != 2 || len(m.FileRepos["pnpm-lock.yaml"]) != 3 {
		t.Errorf("mapping=%+v", m)
	}
	if !reflect.DeepEqual(m.ModuleRepos, []string{"rules_oci+"}) || !reflect.DeepEqual(m.LocalPaths, []string{"../a"}) {
		t.Errorf("mapping=%+v", m)
	}
}

func TestLoadSkipsFailingIDs(t *testing.T) {
	ws := t.TempDir()
	show := read(t, "testdata/show_extension.txt")
	f := &fakeRunner{fn: func(a []string) ([]byte, error) {
		if len(a) > 3 || a[2] == "@@//bad:x.bzl%bad" {
			return nil, errors.New("boom")
		}
		return show, nil
	}}
	mb := []byte(`a = use_extension("@aspect_rules_js//npm:extensions.bzl", "npm")
b = use_extension("//bad:x.bzl", "bad")
`)
	m, warns, err := Load(ws, f, []byte(`{}`), mb)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(warns, []string{"@@//bad:x.bzl%bad"}) {
		t.Errorf("warns=%v", warns)
	}
	if len(m.FileRepos["pnpm-lock.yaml"]) == 0 {
		t.Errorf("good id lost: %+v", m)
	}
	if len(f.calls) != 3 {
		t.Errorf("calls=%v", f.calls)
	}
}

func TestLoadMissingLockfile(t *testing.T) {
	ws := t.TempDir()
	lock := read(t, "testdata/MODULE.bazel.lock")
	f := &fakeRunner{fn: func(a []string) ([]byte, error) {
		if a[0] == "mod" && a[1] == "deps" {
			return nil, os.WriteFile(ws+"/MODULE.bazel.lock", lock, 0o644)
		}
		return nil, nil
	}}
	m, _, err := Load(ws, f, nil, []byte(`module(name="x")`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.calls[0], " "); got != "mod deps --lockfile_mode=update" {
		t.Errorf("call %q", got)
	}
	if len(m.FileRepos["go.sum"]) != 2 {
		t.Errorf("mapping=%+v", m)
	}
}

func TestLoadMissingLockfileFails(t *testing.T) {
	f := &fakeRunner{fn: func(a []string) ([]byte, error) { return nil, errors.New("no bazel") }}
	if _, _, err := Load(t.TempDir(), f, nil, nil); err == nil {
		t.Error("expected error")
	}
}

func TestMatchRepoPrefix(t *testing.T) {
	cases := []struct {
		p, c string
		want bool
	}{
		{"rules_oci++*", "rules_oci++oci+nginx", true},
		{"rules_oci++*", "rules_oci+", false},
		{"rules_oci++*", "other++oci+nginx", false},
		{"*+oci+nginx", "rules_oci++oci+nginx", true},
		{"rules_oci+", "rules_oci+", true},
		{"rules_oci+", "rules_oci++oci+x", false},
	}
	for _, c := range cases {
		if got := MatchRepo(c.p, c.c); got != c.want {
			t.Errorf("MatchRepo(%q,%q)=%v", c.p, c.c, got)
		}
	}
}

func TestModuleChangeExplained(t *testing.T) {
	base := `module(name = "x")
# a comment
bazel_dep(
    name = "rules_oci",  # trailing ) comment
    version = "2.0.0",
)
bazel_dep(name = "zlib", version = "1.0")
oci = use_extension("@rules_oci//oci:extensions.bzl", "oci")
oci.pull(name = "nginx", digest = "sha256:aaa")
use_repo(oci, "nginx")
`
	cases := []struct {
		name string
		edit func(string) string
		want bool
	}{
		{"identical", func(s string) string { return s }, true},
		{"version bump", func(s string) string { return strings.Replace(s, `"2.0.0"`, `"2.1.0"`, 1) }, true},
		{"dep added", func(s string) string { return s + "bazel_dep(name = \"new\", version = \"1\")\n" }, true},
		{"digest", func(s string) string { return strings.Replace(s, "aaa", "bbb", 1) }, false},
		{"git_override", func(s string) string {
			return s + "git_override(module_name = \"zlib\", remote = \"r\", commit = \"c\")\n"
		}, false},
		{"use_repo", func(s string) string {
			return strings.Replace(s, `use_repo(oci, "nginx")`, `use_repo(oci, "nginx", "other")`, 1)
		}, false},
		{"comment outside", func(s string) string { return strings.Replace(s, "# a comment", "# another", 1) }, false},
	}
	for _, c := range cases {
		if got := ModuleChangeExplained([]byte(base), []byte(c.edit(base))); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestMainRepoNames(t *testing.T) {
	mb := "# module(name = \"commented\")\nmodule(\n    name = \"me\",\n    version = \"1\",\n    repo_name = \"alias\",\n)\n"
	if got, want := MainRepoNames([]byte(mb)), []string{"alias", "me"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := RootExtensions([]byte(mb + "a = use_extension(\"@alias//lib:x.bzl\", \"a\")\n")); !reflect.DeepEqual(got, []string{"@@//lib:x.bzl%a"}) {
		t.Errorf("repo_name extension id = %v", got)
	}
	if got := MainRepoNames([]byte("bazel_dep(name = \"x\")\n")); got != nil {
		t.Errorf("no module(): %v", got)
	}
}

func TestModuleBzlPaths(t *testing.T) {
	mb := `module(name = "me", repo_name = "alias")
# x = use_extension("//commented:c.bzl", "x")
a = use_extension("//lib:a.bzl", "a")
b = use_extension("@//lib:b.bzl", "b")
c = use_extension("@@//lib:c.bzl", "c", dev_dependency = True)
d = use_extension("@me//lib:d.bzl", "d")
e = use_extension("@alias//lib:e.bzl", "e")
f = use_extension(":f.bzl", "f")
g = use_extension(extension_name = "g", extension_bzl_file = "//lib:g.bzl")
ext = use_extension("@rules_oci//oci:extensions.bzl", "oci")
http = use_repo_rule("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
mine = use_repo_rule(
    "//tools:repo.bzl",
    "my_repo",
)
`
	want := []string{"f.bzl", "lib/a.bzl", "lib/b.bzl", "lib/c.bzl", "lib/d.bzl", "lib/e.bzl", "lib/g.bzl", "tools/repo.bzl"}
	if got := ModuleBzlPaths([]byte(mb), MainRepoNames([]byte(mb))); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

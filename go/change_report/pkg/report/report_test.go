package report

import (
	"reflect"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitops"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/graph"
)

func fc(status byte, p string) gitx.FileChange { return gitx.FileChange{Status: status, Path: p} }

func commit(sha string, files ...gitx.FileChange) gitx.Commit {
	return gitx.Commit{SHA: sha, Short: sha, Parents: []string{sha + "p"}, Files: files}
}

// input builds an Input where every target has baseline "base" and every
// commit is in range. Commits are given newest first.
func input(closures map[string]graph.Closure, commits ...gitx.Commit) Input {
	in := Input{
		Head:      "head",
		Baselines: map[string]string{},
		Closures:  closures,
		Commits:   commits,
		InRange:   map[string]map[string]bool{"base": {}},
		Tracked:   func(string) bool { return false },
		FileAt:    func(string, string) ([]byte, bool) { return nil, false },
	}
	for l := range closures {
		in.Universe = append(in.Universe, l)
		in.Baselines[l] = "base"
	}
	for _, c := range commits {
		in.InRange["base"][c.SHA] = true
	}
	return in
}

func target(t *testing.T, r Report, label string) Target {
	t.Helper()
	for _, tg := range r.Targets {
		if tg.Label == label {
			return tg
		}
	}
	t.Fatalf("target %s not in report: %+v", label, r.Targets)
	return Target{}
}

func expect(t *testing.T, r Report, label string, status Status, commits ...string) Target {
	t.Helper()
	tg := target(t, r, label)
	if tg.Status != status {
		t.Errorf("%s: status %q, want %q", label, tg.Status, status)
	}
	if len(commits) == 0 {
		commits = nil
	}
	if !reflect.DeepEqual(tg.Commits, commits) {
		t.Errorf("%s: commits %v, want %v", label, tg.Commits, commits)
	}
	return tg
}

func cl(files, pkgs, repos []string) graph.Closure {
	n := func(s []string) []string {
		if s == nil {
			return []string{}
		}
		return s
	}
	return graph.Closure{Files: n(files), Packages: n(pkgs), Repos: n(repos)}
}

func TestFileInClosure(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD", "a/a.go"}, []string{"a"}, nil),
		"//b:b": cl([]string{"b/BUILD", "b/b.go"}, []string{"b"}, nil),
	}, commit("c1", fc('M', "a/a.go")))
	r := Build(in)
	tg := expect(t, r, "//a:a", Changed, "c1")
	if !reflect.DeepEqual(tg.Files, []string{"a/a.go"}) || tg.Since != "base" {
		t.Errorf("files=%v since=%q", tg.Files, tg.Since)
	}
	expect(t, r, "//b:b", Unchanged)
	if !r.Baseline || r.Since != "base" || r.Head != "head" {
		t.Errorf("baseline=%v since=%q head=%q", r.Baseline, r.Since, r.Head)
	}
	if _, ok := r.Commits["c1"]; !ok {
		t.Errorf("commit c1 missing from Commits")
	}
}

func TestAddedFileInPackage(t *testing.T) {
	// A new file matched by a glob is not yet in the closure files.
	in := input(map[string]graph.Closure{
		"//a:a":   cl([]string{"a/BUILD"}, []string{"a"}, nil),
		"//a/x:x": cl([]string{"a/x/BUILD"}, []string{"a/x"}, nil),
		"//b:b":   cl([]string{"b/BUILD"}, []string{"b"}, nil),
	}, commit("c1", fc('A', "a/sub/new.txt")), commit("c0", fc('M', "a/new2.txt")))
	r := Build(in)
	expect(t, r, "//a:a", Changed, "c1")
	expect(t, r, "//a/x:x", Unchanged)
	expect(t, r, "//b:b", Unchanged)
}

func TestDeletedFileInPackage(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a":   cl([]string{"a/BUILD"}, []string{"a"}, nil),
		"//ab:ab": cl([]string{"ab/BUILD"}, []string{"ab"}, nil),
	}, commit("c1", fc('D', "a/old.txt")))
	r := Build(in)
	expect(t, r, "//a:a", Changed, "c1")
	expect(t, r, "//ab:ab", Unchanged)
}

func TestRootPackageMatchesAll(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//:r": cl([]string{"BUILD"}, []string{""}, nil),
	}, commit("c1", fc('A', "deep/dir/f.txt")))
	expect(t, Build(in), "//:r", Changed, "c1")
}

func TestRenameAttributesBothPackages(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD", "a/f.txt"}, []string{"a"}, nil),
		"//b:b": cl([]string{"b/BUILD"}, []string{"b"}, nil),
		"//c:c": cl([]string{"c/BUILD"}, []string{"c"}, nil),
	}, commit("c1", fc('D', "a/f.txt"), fc('A', "b/f.txt")))
	r := Build(in)
	expect(t, r, "//a:a", Changed, "c1")
	tb := expect(t, r, "//b:b", Changed, "c1")
	if !reflect.DeepEqual(tb.Files, []string{"b/f.txt"}) {
		t.Errorf("b files %v", tb.Files)
	}
	expect(t, r, "//c:c", Unchanged)
}

func TestEditOutsideClosureUnchanged(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD", "a/a.go"}, []string{"a"}, nil),
	}, commit("c1", fc('M', "a/unused.txt"), fc('M', "README.md")))
	r := Build(in)
	tg := expect(t, r, "//a:a", Unchanged)
	if len(tg.Files) != 0 {
		t.Errorf("files %v", tg.Files)
	}
}

func TestGlobalFile(t *testing.T) {
	for _, f := range []string{".bazelrc", "WORKSPACE", "WORKSPACE.bazel", "WORKSPACE.bzlmod", ".bazelversion",
		".bazelignore", "REPO.bazel", "x/REPO.bazel", "tools/ci.bazelrc", "deps/go.MODULE.bazel", "MODULE.bazel",
		"MODULE.bazel.lock", "ci/custom.cfg", "local/mod/x.go"} {
		in := input(map[string]graph.Closure{
			"//a:a": cl([]string{"a/BUILD"}, []string{"a"}, nil),
			"//b:b": cl([]string{"b/BUILD"}, []string{"b"}, nil),
		}, commit("c1", fc('M', f)))
		in.GlobalGlobs = []string{"ci/*.cfg"}
		in.Mapping.LocalPaths = []string{"./local/mod/"}
		r := Build(in)
		for _, l := range []string{"//a:a", "//b:b"} {
			tg := expect(t, r, l, Changed, "c1")
			if !reflect.DeepEqual(tg.Files, []string{f}) {
				t.Errorf("%s %s: files %v", f, l, tg.Files)
			}
		}
	}
}

func TestUnmappedLockGlobal(t *testing.T) {
	for _, f := range []string{"web/pnpm-lock.yaml", "go.mod", "x/requirements-dev.txt", "Cargo.lock", "charts/c/Chart.yaml"} {
		in := input(map[string]graph.Closure{
			"//a:a": cl([]string{"a/BUILD"}, []string{"a"}, nil),
			"//b:b": cl([]string{"b/BUILD"}, []string{"b"}, []string{"rules_go+"}),
		}, commit("c1", fc('M', f)))
		in.Mapping.FileRepos = map[string][]string{"go.sum": {"gazelle++go_deps+*"}}
		r := Build(in)
		expect(t, r, "//a:a", Changed, "c1")
		expect(t, r, "//b:b", Changed, "c1")
	}
}

func TestMappedLockScoped(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD"}, []string{"a"}, []string{"rules_go+"}),
		"//b:b": cl([]string{"b/BUILD"}, []string{"b"}, []string{"gazelle++go_deps+com_github_x"}),
	}, commit("c1", fc('M', "go.sum")))
	in.Mapping.FileRepos = map[string][]string{"go.sum": {"gazelle++go_deps+*"}}
	r := Build(in)
	expect(t, r, "//a:a", Unchanged)
	tg := expect(t, r, "//b:b", Changed, "c1")
	if !reflect.DeepEqual(tg.Files, []string{"go.sum"}) {
		t.Errorf("files %v", tg.Files)
	}
}

const lockBefore = `{"moduleExtensions":{"@@gazelle+//:ext.bzl%go_deps":{"general":{"generatedRepoSpecs":{"x":{"v":1},"y":{"v":1}}}}}}`
const lockAfter = `{"moduleExtensions":{"@@gazelle+//:ext.bzl%go_deps":{"general":{"generatedRepoSpecs":{"x":{"v":2},"y":{"v":1}}}}}}`

func level2Input(files map[string][2]string, changes ...gitx.FileChange) Input {
	in := input(map[string]graph.Closure{
		"//x:x":   cl([]string{"x/BUILD"}, []string{"x"}, []string{"gazelle++go_deps+x"}),
		"//y:y":   cl([]string{"y/BUILD"}, []string{"y"}, []string{"gazelle++go_deps+y"}),
		"//z:z":   cl([]string{"z/BUILD"}, []string{"z"}, []string{"rules_go+"}),
		"//n:n":   cl([]string{"n/BUILD"}, []string{"n"}, nil),
		"//zl:zl": cl([]string{"zl/BUILD"}, []string{"zl"}, []string{"zlib+"}),
	}, commit("c1", changes...))
	in.Mapping.FileRepos = map[string][]string{"go.sum": {"gazelle++go_deps+*"}}
	in.Tracked = func(p string) bool { return p == "MODULE.bazel.lock" }
	in.FileAt = func(rev, p string) ([]byte, bool) {
		v, ok := files[p]
		if !ok {
			return nil, false
		}
		switch rev {
		case "c1^":
			return []byte(v[0]), true
		case "c1":
			return []byte(v[1]), true
		}
		return nil, false
	}
	return in
}

func TestLevel2(t *testing.T) {
	in := level2Input(map[string][2]string{"MODULE.bazel.lock": {lockBefore, lockAfter}},
		fc('M', "go.sum"), fc('M', "MODULE.bazel.lock"))
	r := Build(in)
	tg := expect(t, r, "//x:x", Changed, "c1")
	if !reflect.DeepEqual(tg.Files, []string{"MODULE.bazel.lock", "go.sum"}) {
		t.Errorf("files %v", tg.Files)
	}
	expect(t, r, "//y:y", Unchanged)
	expect(t, r, "//z:z", Unchanged)
	expect(t, r, "//n:n", Unchanged)
}

func TestLevel2Untracked(t *testing.T) {
	in := level2Input(map[string][2]string{"MODULE.bazel.lock": {lockBefore, lockAfter}},
		fc('M', "go.sum"), fc('M', "MODULE.bazel.lock"))
	in.Tracked = func(string) bool { return false }
	r := Build(in)
	for _, l := range []string{"//x:x", "//y:y", "//z:z", "//n:n"} {
		expect(t, r, l, Changed, "c1") // lockfile is global
	}
}

func TestLevel2NotOK(t *testing.T) {
	in := level2Input(map[string][2]string{"MODULE.bazel.lock": {lockBefore, "{bad"}},
		fc('M', "go.sum"), fc('M', "MODULE.bazel.lock"))
	r := Build(in)
	for _, l := range []string{"//x:x", "//y:y", "//z:z", "//n:n"} {
		expect(t, r, l, Changed, "c1")
	}
}

func TestLevel2EmptyIntersectionFallsBack(t *testing.T) {
	// The lockfile changed only an unrelated repo; go.sum still changed, so
	// it falls back to the mapped set (never zero).
	before := `{"moduleExtensions":{"@@other+//:e.bzl%e":{"general":{"generatedRepoSpecs":{"q":{"v":1}}}}}}`
	after := `{"moduleExtensions":{"@@other+//:e.bzl%e":{"general":{"generatedRepoSpecs":{"q":{"v":2}}}}}}`
	in := level2Input(map[string][2]string{"MODULE.bazel.lock": {before, after}},
		fc('M', "go.sum"), fc('M', "MODULE.bazel.lock"))
	r := Build(in)
	expect(t, r, "//x:x", Changed, "c1")
	expect(t, r, "//y:y", Changed, "c1")
	expect(t, r, "//z:z", Unchanged)
}

func TestLevel2RootCommit(t *testing.T) {
	in := level2Input(map[string][2]string{"MODULE.bazel.lock": {lockBefore, lockAfter}},
		fc('A', "go.sum"), fc('A', "MODULE.bazel.lock"))
	in.Commits[0].Parents = nil
	r := Build(in)
	for _, l := range []string{"//x:x", "//y:y", "//z:z", "//n:n"} {
		expect(t, r, l, Changed, "c1")
	}
}

const regBefore = `{"registryFileHashes":{"https://bcr.bazel.build/modules/zlib/1.3/MODULE.bazel":"a"}}`
const regAfter = `{"registryFileHashes":{"https://bcr.bazel.build/modules/zlib/1.3.1/MODULE.bazel":"b"}}`

func TestModuleBazelExplained(t *testing.T) {
	in := level2Input(map[string][2]string{
		"MODULE.bazel.lock": {regBefore, regAfter},
		"MODULE.bazel":      {`module(name="m")` + "\n" + `bazel_dep(name = "zlib", version = "1.3")`, `module(name="m")` + "\n" + `bazel_dep(name = "zlib", version = "1.3.1")`},
	}, fc('M', "MODULE.bazel"), fc('M', "MODULE.bazel.lock"))
	r := Build(in)
	tg := expect(t, r, "//zl:zl", Changed, "c1")
	if !reflect.DeepEqual(tg.Files, []string{"MODULE.bazel", "MODULE.bazel.lock"}) {
		t.Errorf("files %v", tg.Files)
	}
	for _, l := range []string{"//x:x", "//y:y", "//z:z", "//n:n"} {
		expect(t, r, l, Unchanged)
	}
}

func TestModuleBazelNotExplained(t *testing.T) {
	in := level2Input(map[string][2]string{
		"MODULE.bazel.lock": {regBefore, regAfter},
		"MODULE.bazel":      {`bazel_dep(name = "zlib", version = "1.3")`, `bazel_dep(name = "zlib", version = "1.3.1")` + "\n# comment"},
	}, fc('M', "MODULE.bazel"), fc('M', "MODULE.bazel.lock"))
	r := Build(in)
	for _, l := range []string{"//x:x", "//y:y", "//z:z", "//n:n", "//zl:zl"} {
		expect(t, r, l, Changed, "c1")
	}
	// MODULE.bazel without the lockfile is always global.
	in = level2Input(map[string][2]string{
		"MODULE.bazel": {`bazel_dep(name = "zlib", version = "1.3")`, `bazel_dep(name = "zlib", version = "1.3.1")`},
	}, fc('M', "MODULE.bazel"))
	r = Build(in)
	for _, l := range []string{"//x:x", "//n:n"} {
		expect(t, r, l, Changed, "c1")
	}
}

func TestMergeUnmanaged(t *testing.T) {
	m := commit("m1")
	m.Parents = []string{"p1", "p2"}
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD", "a/a.go"}, []string{""}, nil),
	}, m, commit("c1", fc('M', ".bazelrc")))
	r := Build(in)
	expect(t, r, "//a:a", Changed, "c1")
	if !reflect.DeepEqual(r.Unmanaged, []string{"m1"}) {
		t.Errorf("unmanaged %v", r.Unmanaged)
	}
	if _, ok := r.Commits["m1"]; !ok {
		t.Errorf("merge commit missing from Commits")
	}
}

func TestNoBaseline(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD", "a/a.go"}, []string{"a"}, nil),
		"//b:b": cl([]string{"b/BUILD"}, []string{"b"}, nil),
	}, commit("c1", fc('M', "a/a.go")))
	delete(in.Baselines, "//b:b")
	in.Baselines["//a:a"] = ""
	r := Build(in)
	for _, l := range []string{"//a:a", "//b:b"} {
		tg := expect(t, r, l, Changed)
		if tg.Since != "" {
			t.Errorf("%s since %q", l, tg.Since)
		}
	}
	if r.Baseline || r.Since != "" {
		t.Errorf("baseline=%v since=%q", r.Baseline, r.Since)
	}
}

func TestPerTargetBaseline(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD", "shared.txt"}, []string{"a"}, nil),
		"//b:b": cl([]string{"b/BUILD", "shared.txt"}, []string{"b"}, nil),
	}, commit("c2", fc('M', "shared.txt")), commit("c1", fc('M', "shared.txt")))
	in.Baselines["//a:a"] = "c1"
	in.Baselines["//b:b"] = "c0"
	in.InRange = map[string]map[string]bool{"c1": {"c2": true}, "c0": {"c1": true, "c2": true}}
	r := Build(in)
	ta := expect(t, r, "//a:a", Changed, "c2")
	tb := expect(t, r, "//b:b", Changed, "c2", "c1")
	if ta.Since != "c1" || tb.Since != "c0" {
		t.Errorf("since %q %q", ta.Since, tb.Since)
	}
	if !r.Baseline || r.Since != "" {
		t.Errorf("baseline=%v since=%q", r.Baseline, r.Since)
	}
}

func TestUnknownRangeIsConservative(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD", "a/a.go"}, []string{"a"}, nil),
	}, commit("c1", fc('M', "a/a.go")))
	in.Baselines["//a:a"] = "unknown"
	r := Build(in)
	expect(t, r, "//a:a", Changed, "c1")
	if len(r.Warnings) == 0 {
		t.Error("expected a warning")
	}
}

func TestMissingClosureChanged(t *testing.T) {
	in := input(map[string]graph.Closure{}, commit("c1", fc('M', "README.md")))
	in.Universe = []string{"//a:a"}
	in.Baselines["//a:a"] = "base"
	r := Build(in)
	expect(t, r, "//a:a", Changed, "c1")
	if len(r.Warnings) == 0 {
		t.Error("expected a warning")
	}
}

func TestRemoved(t *testing.T) {
	in := input(map[string]graph.Closure{
		"//a:a": cl([]string{"a/BUILD"}, []string{"a"}, nil),
	})
	in.Gitops = &gitops.Result{Targets: map[string][]string{
		"//a:a":      {"gitops/a.yaml"},
		"//old:gone": {"gitops/z.yaml", "gitops/y.yaml"},
	}}
	r := Build(in)
	if len(r.Targets) != 2 || r.Targets[0].Label != "//a:a" || r.Targets[1].Label != "//old:gone" {
		t.Fatalf("targets %+v", r.Targets)
	}
	expect(t, r, "//a:a", Unchanged)
	tg := expect(t, r, "//old:gone", Removed)
	if !reflect.DeepEqual(tg.Files, []string{"gitops/y.yaml", "gitops/z.yaml"}) {
		t.Errorf("files %v", tg.Files)
	}
}

func TestMatchGlob(t *testing.T) {
	for _, c := range []struct {
		pat, p string
		want   bool
	}{
		{"**/go.sum", "go.sum", true},
		{"**/go.sum", "a/b/go.sum", true},
		{"**/go.sum", "a/b/go.sum.bak", false},
		{"**/requirements*.txt", "x/requirements-dev.txt", true},
		{"**/requirements*.txt", "requirements.txt", true},
		{"**/*.lock", "a/Cargo.lock", true},
		{"ci/*.cfg", "ci/x.cfg", true},
		{"ci/*.cfg", "ci/sub/x.cfg", false},
		{"ci/**", "ci/sub/x.cfg", true},
		{"a/**/b", "a/b", true},
		{"a/**/b", "a/x/y/b", true},
		{"*.txt", "a/b.txt", false},
		{"[", "[", false},
	} {
		if got := MatchGlob(c.pat, c.p); got != c.want {
			t.Errorf("MatchGlob(%q, %q) = %v, want %v", c.pat, c.p, got, c.want)
		}
	}
}

func TestModuleBazelExplainedNoChangedReposGlobal(t *testing.T) {
	// The lockfile was touched but names no changed repo: a bazel_dep edit
	// that the lockfile does not reflect is not trusted.
	in := level2Input(map[string][2]string{
		"MODULE.bazel.lock": {regBefore, regBefore},
		"MODULE.bazel":      {`bazel_dep(name = "zlib", version = "1.3")`, `bazel_dep(name = "zlib", version = "1.3", dev_dependency = True)`},
	}, fc('M', "MODULE.bazel"), fc('M', "MODULE.bazel.lock"))
	r := Build(in)
	for _, l := range []string{"//x:x", "//n:n", "//zl:zl"} {
		tg := expect(t, r, l, Changed, "c1")
		if !reflect.DeepEqual(tg.Files, []string{"MODULE.bazel"}) {
			t.Errorf("%s files %v", l, tg.Files)
		}
	}
}

package graph

import (
	"reflect"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
)

func bf(pkg string) string {
	if pkg == "" {
		return "BUILD"
	}
	return pkg + "/BUILD"
}

func fixture() []bazelq.Node {
	return []bazelq.Node{
		{Label: "@@//pkg:manifests/a.yaml", Kind: "source"},
		{Label: "@@//pkg:manifests/b.yaml", Kind: "source"},
		{Label: "@@//pkg:viamacro", Kind: "rule", RuleClass: "filegroup", Generator: "mymacro", Inputs: []string{"@@//pkg:manifests/a.yaml", "@@//pkg:manifests/b.yaml"}},
		{Label: "@@//sub:g", Kind: "rule", RuleClass: "genrule", Inputs: []string{"@@//pkg:viamacro", "@@bazel_tools//tools/genrule:genrule-setup.sh"}},
		{Label: "@@//sub:o.txt", Kind: "generated", GeneratingRule: "@@//sub:g"},
	}
}

func TestClosure(t *testing.T) {
	g := New(fixture())
	g.SetLoads("pkg/BUILD", []string{"pkg/m.bzl"})
	g.SetLoads("pkg/m.bzl", []string{"lib/n.bzl"})
	c := g.Closure("//sub:g", bf)
	want := Closure{
		Files:    []string{"lib/n.bzl", "pkg/BUILD", "pkg/m.bzl", "pkg/manifests/a.yaml", "pkg/manifests/b.yaml", "sub/BUILD"},
		Packages: []string{"pkg", "sub"},
		Repos:    []string{"bazel_tools"},
		Calls:    []string{"pkg:mymacro", "sub:g"},
		Macros:   []string{"pkg:mymacro"},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got %+v\nwant %+v", c, want)
	}
}

func TestClosureGeneratedAndCycleAndMissing(t *testing.T) {
	g := New([]bazelq.Node{
		{Label: "@@//a:r1", Kind: "rule", Inputs: []string{"@@//a:r2", "@@//m:missing.txt"}},
		{Label: "@@//a:r2", Kind: "rule", Inputs: []string{"@@//a:r1", "@@//b:gen"}},
		{Label: "@@//b:gen", Kind: "generated", GeneratingRule: "@@//b:r3"},
		{Label: "@@//b:r3", Kind: "rule"},
	})
	g.SetLoads("a/BUILD", []string{"a/BUILD"}) // load cycle
	c := g.Closure("//a:r1", bf)
	want := Closure{
		Files:    []string{"a/BUILD", "b/BUILD", "m/BUILD", "m/missing.txt"},
		Packages: []string{"a", "b", "m"},
		Repos:    []string{},
		Calls:    []string{"a:r1", "a:r2", "b:r3"},
		Macros:   []string{},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got %+v\nwant %+v", c, want)
	}
}

func TestReplacePackages(t *testing.T) {
	g := New(fixture())
	g.ReplacePackages([]string{"pkg"}, []bazelq.Node{{Label: "@@//pkg:new.yaml", Kind: "source"}})
	if _, ok := g.Nodes["@@//pkg:viamacro"]; ok {
		t.Error("old pkg node kept")
	}
	if _, ok := g.Nodes["@@//pkg:new.yaml"]; !ok {
		t.Error("new node missing")
	}
	if _, ok := g.Nodes["@@//sub:g"]; !ok {
		t.Error("other pkg node dropped")
	}
}

func TestParseLoads(t *testing.T) {
	src := []byte(`load(":m.bzl", "x")
load(
  "//lib:a.bzl",
  "y",
)
load("@x//:y.bzl", "z")
load("@@x//:y.bzl", "z")
load("@self//lib:s.bzl", "s")
load("@@self//lib:t.bzl", "s")
load("@//lib:u.bzl", "s")
load("//:root.bzl", "r")
`)
	got := ParseLoads(src, "pkg/sub", []string{"self"})
	want := []string{"pkg/sub/m.bzl", "lib/a.bzl", "lib/s.bzl", "lib/t.bzl", "lib/u.bzl", "root.bzl"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
	got = ParseLoads([]byte(`load(":m.bzl")`), "", nil)
	if !reflect.DeepEqual(got, []string{"m.bzl"}) {
		t.Errorf("root: %v", got)
	}
}

// A relative load in a .bzl file below its package directory resolves against
// the package, not the file's directory.
func TestParseLoadsRelativeToPackage(t *testing.T) {
	// pkg/defs/x.bzl lives in package "pkg" (no pkg/defs/BUILD).
	got := ParseLoads([]byte(`load(":y.bzl", "y")`), "pkg", nil)
	if want := []string{"pkg/y.bzl"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseLoadsWhitespaceAndWordBoundary(t *testing.T) {
	src := []byte("load (\"//a:b.bzl\", \"b\")\nload\t(\"//a:c.bzl\", \"c\")\nautoload(\"//a:no.bzl\")\n")
	got := ParseLoads(src, "", nil)
	if want := []string{"a/b.bzl", "a/c.bzl"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLoadClosure(t *testing.T) {
	g := New(nil)
	g.SetLoads("ext.bzl", []string{"lib/a.bzl"})
	g.SetLoads("lib/a.bzl", []string{"lib/b.bzl", "ext.bzl"})
	got := g.LoadClosure([]string{"ext.bzl", "other.bzl"})
	if want := []string{"ext.bzl", "lib/a.bzl", "lib/b.bzl", "other.bzl"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

package bazelq

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

type fakeRunner struct {
	args [][]string
	out  string
	err  error
}

func (f *fakeRunner) Run(args ...string) ([]byte, error) {
	f.args = append(f.args, args)
	return []byte(f.out), f.err
}

func TestParse(t *testing.T) {
	f, err := os.Open("testdata/deps.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	nodes, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	want := []Node{
		{Label: "@@//pkg:manifests/a.yaml", Kind: "source"},
		{Label: "@@//pkg:manifests/b.yaml", Kind: "source"},
		{Label: "@@//pkg:viamacro", Kind: "rule", RuleClass: "filegroup", Inputs: []string{"@@//pkg:manifests/a.yaml", "@@//pkg:manifests/b.yaml"}},
		{Label: "@@//sub:g", Kind: "rule", RuleClass: "genrule", Inputs: []string{"@@//pkg:viamacro", "@@bazel_tools//tools/genrule:genrule-setup.sh"}},
		{Label: "@@//sub:o.txt", Kind: "generated", GeneratingRule: "@@//sub:g"},
	}
	if !reflect.DeepEqual(nodes, want) {
		t.Errorf("got %+v\nwant %+v", nodes, want)
	}
}

func TestParseBadJSON(t *testing.T) {
	if _, err := Parse(strings.NewReader("{not json\n")); err == nil {
		t.Error("expected error")
	}
}

func TestParseLargeLine(t *testing.T) {
	line := `{"type":"SOURCE_FILE","sourceFile":{"name":"@@//a:b","x":"` + strings.Repeat("x", 300000) + `"}}`
	nodes, err := Parse(strings.NewReader(line + "\n"))
	if err != nil || len(nodes) != 1 {
		t.Fatalf("nodes=%v err=%v", nodes, err)
	}
}

func TestQueryArgs(t *testing.T) {
	f := &fakeRunner{}
	if _, err := Query(f, "deps(//a:b)"); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(f.args[0], " ")
	for _, w := range []string{"query", "--consistent_labels", "--output=streamed_jsonproto", "--keep_going", "deps(//a:b)"} {
		if !strings.Contains(got, w) {
			t.Errorf("args %q missing %q", got, w)
		}
	}
}

func TestQueryError(t *testing.T) {
	if _, err := Query(&fakeRunner{err: errors.New("exit status 3")}, "x"); err == nil {
		t.Error("expected error")
	}
}

func TestLabelsSorted(t *testing.T) {
	f := &fakeRunner{out: "//b:b\n//a:a\n\n"}
	got, err := Labels(f, "//...")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"//a:a", "//b:b"}) {
		t.Errorf("got %v", got)
	}
	if !strings.Contains(strings.Join(f.args[0], " "), "--consistent_labels") {
		t.Error("missing --consistent_labels")
	}
}

func TestVersion(t *testing.T) {
	v, err := Version(&fakeRunner{out: "Bazelisk version: v1\nBuild label: 9.3.1\nBuild time: x\n"})
	if err != nil || v != "9.3.1" {
		t.Errorf("v=%q err=%v", v, err)
	}
	if _, err := Version(&fakeRunner{out: "nothing\n"}); err == nil {
		t.Error("expected error")
	}
}

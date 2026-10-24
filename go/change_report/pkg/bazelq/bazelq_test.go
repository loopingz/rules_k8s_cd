package bazelq

import (
	"errors"
	"fmt"
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
	for i := range nodes {
		nodes[i].Attrs = nil // covered by TestParseAttrs
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

type exitErr int

func (e exitErr) Error() string { return "exit status " + string(rune('0'+int(e))) }
func (e exitErr) ExitCode() int { return int(e) }

func TestExistingLabels(t *testing.T) {
	f := &fakeRunner{out: "@@//a:a\n", err: fmtWrap(exitErr(3))}
	got, err := ExistingLabels(f, []string{"//a:a", "//gone:x"})
	if err != nil {
		t.Fatalf("exit 3 must be accepted: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"@@//a:a"}) {
		t.Errorf("got %v", got)
	}
	args := strings.Join(f.args[0], " ")
	for _, w := range []string{"query", "--keep_going", "--consistent_labels", "--output=label", `set("//a:a" "//gone:x")`} {
		if !strings.Contains(args, w) {
			t.Errorf("args %q missing %q", args, w)
		}
	}

	if got, err := ExistingLabels(&fakeRunner{out: "//b:b\n"}, []string{"//b:b"}); err != nil || !reflect.DeepEqual(got, []string{"//b:b"}) {
		t.Errorf("exit 0: got %v, %v", got, err)
	}
	if _, err := ExistingLabels(&fakeRunner{out: "//b:b\n", err: fmtWrap(exitErr(7))}, []string{"//b:b"}); err == nil {
		t.Error("exit 7 must be an error")
	}
	if _, err := ExistingLabels(&fakeRunner{err: errors.New("boom")}, []string{"//b:b"}); err == nil {
		t.Error("plain failure must be an error")
	}
	if _, err := ExistingLabels(&fakeRunner{}, []string{`//b:"x`}); err == nil {
		t.Error("unquotable label must be an error")
	}
}

func fmtWrap(err error) error { return fmt.Errorf("bazel query: %w: stderr", err) }

func TestParseAttrs(t *testing.T) {
	f, err := os.Open("testdata/attrs.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	nodes, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	a := nodes[0].Attrs
	if got := a["s"]; got.Type != "STRING" || got.String != "hello" {
		t.Errorf("s = %+v", got)
	}
	if got := a["sl"]; got.Type != "STRING_LIST" || !reflect.DeepEqual(got.StringList, []string{"a", "b"}) {
		t.Errorf("sl = %+v", got)
	}
	if got := a["ll"]; got.Type != "LABEL_LIST" || !reflect.DeepEqual(got.StringList, []string{"@@//pkg:y", "@@//pkg:z"}) {
		t.Errorf("ll = %+v", got)
	}
	if got := a["sd"]; got.Type != "STRING_DICT" || !reflect.DeepEqual(got.StringDict, map[string]string{"k1": "v1", "k2": "v2"}) {
		t.Errorf("sd = %+v", got)
	}
	if got := a["b"]; got.Type != "BOOLEAN" || !got.Bool {
		t.Errorf("b = %+v", got)
	}
	if _, ok := a["i"]; !ok {
		t.Error("explicit INTEGER attr i should be recorded (type only)")
	} else if a["i"].Type != "INTEGER" {
		t.Errorf("i = %+v", a["i"])
	}
	if _, ok := a["exec_group_compatible_with"]; ok {
		t.Error("non-explicit LABEL_LIST_DICT should be skipped")
	}
	if got, ok := a["tags"]; !ok || got.Type != "STRING_LIST" || got.StringList != nil {
		t.Errorf("tags = %+v, %v", got, ok)
	}
}

func TestParseAttrsNonRule(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`{"type":"SOURCE_FILE","sourceFile":{"name":"@@//a:b.txt"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if nodes[0].Attrs != nil {
		t.Errorf("source node has Attrs %v", nodes[0].Attrs)
	}
}

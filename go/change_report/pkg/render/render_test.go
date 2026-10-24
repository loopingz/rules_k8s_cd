package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
)

var update = flag.Bool("update", false, "regenerate golden files")

const (
	shaBase  = "aaaaaaa0000000000000000000000000000000aa"
	shaHead  = "ddddddd00000000000000000000000000000ddd0"
	shaC1    = "1111111000000000000000000000000000000111"
	shaC2    = "2222222000000000000000000000000000000222"
	shaMerge = "3333333000000000000000000000000000000333"
)

func fixture() report.Report {
	d := time.Date(2026, 10, 1, 12, 30, 0, 0, time.UTC)
	return report.Report{
		Since:    shaBase,
		Head:     shaHead,
		Baseline: true,
		Targets: []report.Target{
			{Label: "//apps/api:deploy", Status: report.Changed, Since: shaBase, Commits: []string{shaC2, shaC1}, Files: []string{"apps/api/main.go", "apps/api/values.yaml"}},
			{Label: "//apps/old:deploy", Status: report.Removed, Since: shaBase, Files: []string{"gitops/old/a.yaml", "gitops/old/b.yaml"}},
			{Label: "//apps/web:deploy", Status: report.Changed, Since: shaBase, Commits: []string{shaC1}, Files: []string{"apps/web/index.html"}},
			{Label: "//apps/worker:deploy", Status: report.Unchanged, Since: shaBase},
		},
		Unmanaged: []string{shaMerge},
		Commits: map[string]gitx.Commit{
			shaHead:  {SHA: shaHead, Short: "ddddddd", Subject: "unrelated head"},
			shaC2:    {SHA: shaC2, Short: "2222222", Author: "Bob", Email: "bob@example.com", Date: d.Add(time.Hour), Subject: "fix: handle a|b <nil>", Body: "Longer explanation.\n\nSecond paragraph.\n</details> tricky"},
			shaC1:    {SHA: shaC1, Short: "1111111", Author: "Alice", Email: "alice@example.com", Date: d, Subject: "feat: add things"},
			shaMerge: {SHA: shaMerge, Short: "3333333", Author: "Carol", Email: "carol@example.com", Date: d.Add(2 * time.Hour), Subject: "Merge branch 'x' <into> main"},
		},
	}
}

const url = "https://example.com/commit/{sha}"

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(p, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Errorf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func TestMarkdownGolden(t *testing.T) {
	var b bytes.Buffer
	if err := Markdown(&b, fixture(), url); err != nil {
		t.Fatal(err)
	}
	golden(t, "report.md.golden", b.Bytes())
}

func TestJSONGolden(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, fixture(), url); err != nil {
		t.Fatal(err)
	}
	golden(t, "report.json.golden", b.Bytes())
}

func TestMarkdownNoBaseline(t *testing.T) {
	r := report.Report{Head: shaHead, Targets: []report.Target{{Label: "//a:b", Status: report.Changed}}, Commits: map[string]gitx.Commit{}}
	var b bytes.Buffer
	if err := Markdown(&b, r, ""); err != nil {
		t.Fatal(err)
	}
	golden(t, "nobaseline.md.golden", b.Bytes())
}

func TestMarkdownMixed(t *testing.T) {
	r := fixture()
	r.Baseline, r.Since = false, ""
	r.Targets[2].Since = ""
	r.Targets[2].Commits = nil
	var b bytes.Buffer
	if err := Markdown(&b, r, url); err != nil {
		t.Fatal(err)
	}
	golden(t, "mixed.md.golden", b.Bytes())
}

func TestEscapeCells(t *testing.T) {
	if got := escapeCell("a|b <c>\nd"); got != `a\|b &lt;c&gt; d` {
		t.Errorf("got %q", got)
	}
}

func TestNoURL(t *testing.T) {
	var b bytes.Buffer
	if err := Markdown(&b, fixture(), ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "](") {
		t.Errorf("link emitted without template:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "| 1111111 | feat: add things | Alice |") {
		t.Errorf("plain short row missing:\n%s", b.String())
	}
	b.Reset()
	if err := JSON(&b, fixture(), ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), `"url"`) {
		t.Errorf("url present:\n%s", b.String())
	}
}

func TestCommitURL(t *testing.T) {
	if got := CommitURL("https://x/{sha}/y", "abc"); got != "https://x/abc/y" {
		t.Error(got)
	}
	if CommitURL("", "abc") != "" {
		t.Error("empty template must give empty url")
	}
}

func TestTargets(t *testing.T) {
	var b bytes.Buffer
	if err := Targets(&b, fixture()); err != nil {
		t.Fatal(err)
	}
	if want := "//apps/api:deploy\n//apps/web:deploy\n"; b.String() != want {
		t.Errorf("got %q", b.String())
	}
}

func TestRemoved(t *testing.T) {
	var b bytes.Buffer
	if err := Removed(&b, fixture()); err != nil {
		t.Fatal(err)
	}
	if want := "gitops/old/a.yaml\ngitops/old/b.yaml\n"; b.String() != want {
		t.Errorf("got %q", b.String())
	}
}

func TestEmptyReport(t *testing.T) {
	r := report.Report{Head: shaHead, Baseline: true, Since: shaBase}
	var b bytes.Buffer
	if err := JSON(&b, r, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"targets": []`) || !strings.Contains(b.String(), `"unmanaged_commits": []`) {
		t.Errorf("nulls in output:\n%s", b.String())
	}
	b.Reset()
	if err := Markdown(&b, r, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "0 target(s) changed.") {
		t.Errorf("got %s", b.String())
	}
}

package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/release/pkg/conventional"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

var day = time.Date(2026, 10, 24, 23, 30, 0, 0, time.FixedZone("x", -5*3600))

func entry(i int, subject, body string, via ...string) Entry {
	c := gitx.Commit{SHA: fmt.Sprintf("%040d", i), Short: fmt.Sprintf("%07d", i), Subject: subject, Body: body}
	return Entry{Commit: c, Parsed: conventional.Parse(c), Via: via}
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatalf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func TestSectionGolden(t *testing.T) {
	entries := []Entry{
		entry(1, "feat(api)!: drop v1", ""),
		entry(2, "feat: newest feature", "", "lib/x.go"),
		entry(3, "feat(ui): older feature", "", "a.go", "b.go", "c.go"),
		entry(4, "fix: newest fix", ""),
		entry(5, "perf: faster", ""),
		entry(6, "Merge branch 'x'", ""),
		entry(7, "chore: tidy", ""),
		entry(8, "fix: older fix", "BREAKING CHANGE: oops"),
	}
	v := semver.Version{Major: 2, Minor: 0, Patch: 0}
	golden(t, "section.golden", Section(v, day, entries, "", false))
	golden(t, "section_url.golden", Section(v, day, entries[:2], "https://example.com/c/{sha}", false))
}

func TestSectionDateUTC(t *testing.T) {
	s := Section(semver.Version{Major: 1}, day, nil, "", false)
	if s != "## 1.0.0 (2026-10-25)\n" {
		t.Fatalf("got %q", s)
	}
}

func TestSectionEscaping(t *testing.T) {
	s := Section(semver.Version{Major: 1}, day, []Entry{
		entry(1, "feat(<s>*): handle <script> & #1", ""),
		entry(2, "# heading-like", ""),
		entry(3, "fix: * star", "", "a`b.go"),
	}, "", false)
	for _, bad := range []string{"<script>", "<s>", "- # head", "- * star"} {
		if strings.Contains(s, bad) {
			t.Errorf("unescaped %q in\n%s", bad, s)
		}
	}
	for _, good := range []string{"&lt;script&gt;", `**&lt;s&gt;\*:**`, `- \# heading-like`, `- \* star`, "via `a'b.go`"} {
		if !strings.Contains(s, good) {
			t.Errorf("missing %q in\n%s", good, s)
		}
	}
}

func TestFirstReleaseGolden(t *testing.T) {
	var entries []Entry
	for i := 60; i >= 1; i-- {
		entries = append(entries, entry(i, fmt.Sprintf("feat: change %d", i), ""))
	}
	got := Section(semver.Version{Minor: 1}, day, entries, "", true)
	golden(t, "first.golden", got)
	if !strings.Contains(got, "… and 10 earlier commits") || strings.Contains(got, "change 10\n") {
		t.Fatalf("bad truncation:\n%s", got)
	}
}

func TestFirstReleaseSmall(t *testing.T) {
	got := Section(semver.Version{Minor: 1}, day, []Entry{entry(1, "fix: a", "")}, "", true)
	if !strings.Contains(got, "Initial release") || strings.Contains(got, "earlier") {
		t.Fatalf("got %s", got)
	}
}

func TestPrependRealChangelog(t *testing.T) {
	existing, err := os.ReadFile("testdata/CHANGELOG.head.md")
	if err != nil {
		t.Fatal(err)
	}
	sec := Section(semver.Version{Major: 4, Minor: 2}, day, []Entry{entry(1, "feat: x", "")}, "", false)
	got := string(Prepend(existing, sec))
	golden(t, "prepend.golden", got)
	if !strings.HasPrefix(got, "# Changelog\n\n## 4.2.0 (") || !strings.Contains(got, "\n\n## [4.1.0]") {
		t.Fatalf("bad layout:\n%s", got)
	}
	again := string(Prepend([]byte(got), Section(semver.Version{Major: 4, Minor: 3}, day, nil, "", false)))
	if strings.Count(again, "\n\n\n## ") != strings.Count(got, "\n\n\n## ") || !strings.Contains(again, "## 4.3.0 (2026-10-25)\n\n## 4.2.0") {
		t.Fatalf("bad second prepend:\n%s", again)
	}
}

func TestPrependEmptyAndNoTitle(t *testing.T) {
	sec := "## 1.0.0 (2026-10-24)\n"
	if got := string(Prepend(nil, sec)); got != sec {
		t.Fatalf("got %q", got)
	}
	if got := string(Prepend([]byte("\n\n"), sec)); got != sec {
		t.Fatalf("got %q", got)
	}
	if got := string(Prepend([]byte("## 0.9.0\n\nold\n"), sec)); got != sec+"\n## 0.9.0\n\nold\n" {
		t.Fatalf("got %q", got)
	}
	if got := string(Prepend([]byte("# Changelog\n"), sec)); got != "# Changelog\n\n"+sec {
		t.Fatalf("got %q", got)
	}
	if got := string(Prepend([]byte("# Changelog\n\nAll notable changes.\n\n## 0.9.0\n"), sec)); got != "# Changelog\n\nAll notable changes.\n\n"+sec+"\n## 0.9.0\n" {
		t.Fatalf("got %q", got)
	}
}

func TestExtract(t *testing.T) {
	existing, err := os.ReadFile("testdata/CHANGELOG.head.md")
	if err != nil {
		t.Fatal(err)
	}
	got := Extract(existing, semver.Version{Major: 4, Minor: 1})
	if !strings.HasPrefix(got, "### Features") || !strings.Contains(got, "### Bug Fixes") || strings.Contains(got, "4.0.3") || strings.Contains(got, "\n## ") {
		t.Fatalf("got %q", got)
	}
	if Extract(existing, semver.Version{Major: 9}) != "" || Extract(existing, semver.Version{Major: 4, Minor: 1, Patch: 0}) == "" {
		t.Fatal("presence")
	}
	if Extract(existing, semver.Version{Major: 4, Minor: 0, Patch: 3}) == "" {
		t.Fatal("4.0.3 missing")
	}

	sec := Section(semver.Version{Major: 2}, day, []Entry{entry(1, "feat: x", "")}, "", false)
	file := Prepend([]byte("# Changelog\n\n## 1.0.0 (2026-01-01)\n\nold\n"), sec)
	want := "### Features\n\n- x (0000001)"
	if got := Extract(file, semver.Version{Major: 2}); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := Extract(file, semver.Version{Major: 1}); got != "old" {
		t.Fatalf("got %q", got)
	}
	if Extract(file, semver.Version{Major: 2, Minor: 0, Patch: 1}) != "" {
		t.Fatal("prefix confusion")
	}
}

func TestPRBody(t *testing.T) {
	got := PRBody("app", "## 1.0.0 (d)\n")
	if got != "<!-- rules_k8s_cd:release app -->\n\n## 1.0.0 (d)\n" {
		t.Fatalf("got %q", got)
	}
}

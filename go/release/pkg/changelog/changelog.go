// Package changelog renders release changelog sections and merges them into
// CHANGELOG.md files.
package changelog

import (
	"fmt"
	"strings"
	"time"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/release/pkg/conventional"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

// Entry is one commit attributed to a component.
type Entry struct {
	Commit gitx.Commit
	Parsed conventional.Parsed
	// Via lists attributed files outside the component's own packages.
	Via []string
}

// maxFirst is the number of commits listed in a first-release changelog.
const maxFirst = 50

var groups = []string{"Breaking Changes", "Features", "Bug Fixes", "Other"}

func groupOf(p conventional.Parsed) int {
	switch {
	case p.Breaking:
		return 0
	case !p.Conventional:
		return 3
	}
	switch strings.ToLower(p.Type) {
	case "feat":
		return 1
	case "fix", "perf", "revert":
		return 2
	}
	return 3
}

var angle = strings.NewReplacer("<", "&lt;", ">", "&gt;")

// escapeText makes s safe inside a Markdown list item: it is single-line,
// has no raw HTML and no leading character that Markdown would interpret.
func escapeText(s string) string {
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
	s = angle.Replace(strings.TrimSpace(s))
	if s != "" && strings.ContainsRune("#*+-_`[|!", rune(s[0])) {
		s = `\` + s
	}
	return s
}

func escapeScope(s string) string {
	s = strings.NewReplacer("*", `\*`, "\r", " ", "\n", " ").Replace(s)
	return angle.Replace(s)
}

func line(e Entry, commitURL string) string {
	var b strings.Builder
	b.WriteString("- ")
	if e.Parsed.Scope != "" {
		fmt.Fprintf(&b, "**%s:** ", escapeScope(e.Parsed.Scope))
	}
	b.WriteString(escapeText(e.Parsed.Description))
	short := e.Commit.Short
	if short == "" && len(e.Commit.SHA) >= 7 {
		short = e.Commit.SHA[:7]
	}
	if short != "" {
		if commitURL != "" {
			fmt.Fprintf(&b, " ([%s](%s))", short, strings.ReplaceAll(commitURL, "{sha}", e.Commit.SHA))
		} else {
			fmt.Fprintf(&b, " (%s)", short)
		}
	}
	if len(e.Via) > 0 {
		fmt.Fprintf(&b, " — via `%s`", strings.ReplaceAll(e.Via[0], "`", "'"))
		if n := len(e.Via) - 1; n > 0 {
			fmt.Fprintf(&b, " +%d more", n)
		}
	}
	return b.String()
}

// Section renders the changelog section for version. entries are newest
// first and are listed in that order within each group. commitURL is a
// template with a {sha} placeholder (or empty). A first release lists only
// the 50 most recent entries and counts the rest.
func Section(version semver.Version, date time.Time, entries []Entry, commitURL string, first bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%s)\n", version, date.UTC().Format("2006-01-02"))
	earlier := 0
	if first {
		b.WriteString("\nInitial release\n")
		if len(entries) > maxFirst {
			earlier = len(entries) - maxFirst
			entries = entries[:maxFirst]
		}
	}
	for gi, name := range groups {
		var lines []string
		for _, e := range entries {
			if groupOf(e.Parsed) == gi {
				lines = append(lines, line(e, commitURL))
			}
		}
		if len(lines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n### %s\n\n%s\n", name, strings.Join(lines, "\n"))
	}
	if earlier > 0 {
		fmt.Fprintf(&b, "\n… and %d earlier commits\n", earlier)
	}
	return b.String()
}

// Prepend inserts section into existing, after a leading "# ..." title block
// if there is one, otherwise at the top, separated by one blank line.
func Prepend(existing []byte, section string) []byte {
	section = strings.TrimRight(section, "\n") + "\n"
	text := strings.ReplaceAll(string(existing), "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		return []byte(section)
	}
	head := ""
	rest := strings.TrimLeft(text, "\n")
	if strings.HasPrefix(rest, "# ") {
		idx := len(rest)
		for off, l := 0, ""; off < len(rest); off += len(l) + 1 {
			l = rest[off:]
			if i := strings.IndexByte(l, '\n'); i >= 0 {
				l = l[:i]
			}
			if off > 0 && strings.HasPrefix(l, "## ") {
				idx = off
				break
			}
		}
		head = strings.TrimRight(rest[:idx], "\n") + "\n\n"
		rest = rest[idx:]
	}
	rest = strings.TrimLeft(rest, "\n")
	out := head + section
	if rest != "" {
		out += "\n" + rest
	}
	return []byte(out)
}

// Extract returns the body of the section for v (heading "## X.Y.Z..." or
// "## [X.Y.Z]...") up to the next "## " heading, trimmed; "" if absent.
func Extract(content []byte, v semver.Version) string {
	plain, linked := "## "+v.String(), "## ["+v.String()+"]"
	var body []string
	in := false
	for _, l := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(l, "## ") {
			if in {
				break
			}
			in = headingIs(l, plain) || headingIs(l, linked)
			continue
		}
		if in {
			body = append(body, l)
		}
	}
	return strings.TrimSpace(strings.Join(body, "\n"))
}

func headingIs(l, prefix string) bool {
	if !strings.HasPrefix(l, prefix) {
		return false
	}
	rest := l[len(prefix):]
	return rest == "" || rest[0] == ' ' || rest[0] == '('
}

// Marker is the line identifying the release PR of component name; only a
// PR whose body carries it is updated or closed by the release tool.
func Marker(name string) string {
	return fmt.Sprintf("<!-- rules_k8s_cd:release %s -->", name)
}

// PRBody returns the release PR body: the marker that identifies the
// component's release PR followed by section.
func PRBody(name string, section string) string {
	return Marker(name) + "\n\n" + section
}

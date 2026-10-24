// Package render turns a report.Report into the CLI's output formats: a
// target list, a removed-file list, JSON and a Markdown changelog.
package render

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
)

// CommitURL expands the {sha} placeholder of tmpl. An empty template yields "".
func CommitURL(tmpl, sha string) string {
	if tmpl == "" {
		return ""
	}
	return strings.ReplaceAll(tmpl, "{sha}", sha)
}

// Targets writes the labels of changed targets, sorted, one per line.
func Targets(w io.Writer, r report.Report) error {
	var labels []string
	for _, t := range r.Targets {
		if t.Status == report.Changed {
			labels = append(labels, t.Label)
		}
	}
	sort.Strings(labels)
	return writeLines(w, labels)
}

// Removed writes the files of removed targets, sorted, one per line.
func Removed(w io.Writer, r report.Report) error {
	var files []string
	for _, t := range r.Targets {
		if t.Status == report.Removed {
			files = append(files, t.Files...)
		}
	}
	sort.Strings(files)
	return writeLines(w, files)
}

func writeLines(w io.Writer, lines []string) error {
	bw := bufio.NewWriter(w)
	for _, l := range lines {
		bw.WriteString(l)
		bw.WriteByte('\n')
	}
	return bw.Flush()
}

type jsonTarget struct {
	Label   string   `json:"label"`
	Status  string   `json:"status"`
	Since   string   `json:"since"`
	Commits []string `json:"commits"`
	Files   []string `json:"files"`
}

type jsonCommit struct {
	Short   string `json:"short"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	URL     string `json:"url,omitempty"`
}

type jsonReport struct {
	Since            string                `json:"since"`
	Head             string                `json:"head"`
	Baseline         bool                  `json:"baseline"`
	Targets          []jsonTarget          `json:"targets"`
	UnmanagedCommits []string              `json:"unmanaged_commits"`
	Commits          map[string]jsonCommit `json:"commits"`
}

// JSON writes the report as indented JSON. commitURL is a template with a
// {sha} placeholder; the url field is omitted when it is empty.
func JSON(w io.Writer, r report.Report, commitURL string) error {
	out := jsonReport{
		Since:            r.Since,
		Head:             r.Head,
		Baseline:         r.Baseline,
		Targets:          []jsonTarget{},
		UnmanagedCommits: orEmpty(r.Unmanaged),
		Commits:          map[string]jsonCommit{},
	}
	used := func(sha string) {
		c := lookup(r, sha)
		out.Commits[sha] = jsonCommit{
			Short:   shortOf(c),
			Author:  c.Author,
			Email:   c.Email,
			Date:    c.Date.Format(time.RFC3339),
			Subject: c.Subject,
			Body:    c.Body,
			URL:     CommitURL(commitURL, sha),
		}
	}
	for _, t := range r.Targets {
		out.Targets = append(out.Targets, jsonTarget{
			Label: t.Label, Status: string(t.Status), Since: t.Since,
			Commits: orEmpty(t.Commits), Files: orEmpty(t.Files),
		})
		for _, sha := range t.Commits {
			used(sha)
		}
	}
	for _, sha := range r.Unmanaged {
		used(sha)
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func lookup(r report.Report, sha string) gitx.Commit {
	if c, ok := r.Commits[sha]; ok {
		return c
	}
	return gitx.Commit{SHA: sha}
}

func short7(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func shortOf(c gitx.Commit) string {
	if c.Short != "" {
		return c.Short
	}
	return short7(c.SHA)
}

func shortSHA(r report.Report, sha string) string {
	return shortOf(lookup(r, sha))
}

var angle = strings.NewReplacer("<", "&lt;", ">", "&gt;")

// escapeCell makes s safe inside a Markdown table cell.
func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
	return angle.Replace(s)
}

// escapeBody keeps a commit body from closing the surrounding <details>.
func escapeBody(s string) string {
	return strings.ReplaceAll(s, "</details>", "&lt;/details&gt;")
}

// Markdown writes the changelog used as a GitOps PR body.
func Markdown(w io.Writer, r report.Report, commitURL string) error {
	bw := bufio.NewWriter(w)
	p := func(format string, a ...any) { fmt.Fprintf(bw, format, a...) }

	headShort := short7(r.Head)
	if c, ok := r.Commits[r.Head]; ok {
		headShort = shortOf(c)
	}
	anySince := false
	for _, t := range r.Targets {
		if t.Status != report.Removed && t.Since != "" {
			anySince = true
		}
	}
	shared := r.Baseline && r.Since != ""
	switch {
	case shared && r.Since == report.RootBaseline:
		p("# Changes (all history)..%s\n", headShort)
	case shared:
		p("# Changes %s..%s\n", shortSHA(r, r.Since), headShort)
	case anySince:
		p("# Changes up to %s\n", headShort)
	default:
		p("# Changes (no baseline: full deployment)\n")
	}

	var changed, removed []report.Target
	for _, t := range r.Targets {
		switch t.Status {
		case report.Changed:
			changed = append(changed, t)
		case report.Removed:
			removed = append(removed, t)
		}
	}
	p("\n%d target(s) changed.\n", len(changed))

	link := func(sha string) string {
		s := shortSHA(r, sha)
		if u := CommitURL(commitURL, sha); u != "" {
			return fmt.Sprintf("[%s](%s)", s, u)
		}
		return s
	}

	for _, t := range changed {
		p("\n## %s\n\n", t.Label)
		if t.Since == "" {
			p("_No baseline: full deployment._\n")
			continue
		}
		if !shared {
			p("_Since %s_\n\n", shortSHA(r, t.Since))
		}
		if len(t.Commits) == 0 {
			continue
		}
		p("| Commit | Subject | Author |\n|---|---|---|\n")
		for _, sha := range t.Commits {
			c := lookup(r, sha)
			p("| %s | %s | %s |\n", link(sha), escapeCell(c.Subject), escapeCell(c.Author))
		}
		p("\n<details><summary>Full commit messages</summary>\n")
		for _, sha := range t.Commits {
			c := lookup(r, sha)
			p("\n### %s — %s\n", shortOf(c), angle.Replace(c.Subject))
			if c.Body != "" {
				p("\n%s\n", escapeBody(c.Body))
			}
		}
		p("\n</details>\n")
	}

	if len(removed) > 0 {
		p("\n## Removed\n\n")
		for _, t := range removed {
			p("- %s\n", t.Label)
			for _, f := range t.Files {
				p("  - %s\n", f)
			}
		}
	}

	if len(r.Unmanaged) > 0 {
		p("\n## Not analysed\n\nMerge commits are not attributed to targets:\n\n")
		for _, sha := range r.Unmanaged {
			p("- %s %s\n", shortSHA(r, sha), angle.Replace(lookup(r, sha).Subject))
		}
	}
	return bw.Flush()
}

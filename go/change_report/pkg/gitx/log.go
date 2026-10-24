package gitx

import (
	"fmt"
	"strings"
	"time"
)

// FileChange is one file touched by a commit. Status is 'A', 'M', 'D' or 'T'.
type FileChange struct {
	Status byte
	Path   string
}

// Commit is a parsed git commit. Body excludes the subject line and is trimmed.
type Commit struct {
	SHA, Short, Author, Email string
	Date                      time.Time
	Subject, Body             string
	Parents                   []string
	Files                     []FileChange
}

// Header fields are NUL separated; each record starts with 0x1e. Merge
// commits get no name-status output from git log by default, so Files is nil.
const logFormat = "--format=%x1e%H%x00%h%x00%an%x00%ae%x00%aI%x00%P%x00%s%x00%b%x00"

const headerFields = 8

// Log returns the commits in from..to (all history up to to when from is
// empty), newest first.
func (r Repo) Log(from, to string) ([]Commit, error) {
	out, err := r.run("log", "--name-status", "--no-renames", "-z", logFormat, rangeSpec(from, to), "--")
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, rec := range strings.Split(string(out), "\x1e") {
		if rec == "" {
			continue
		}
		c, err := parseRecord(rec)
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func parseRecord(rec string) (Commit, error) {
	f := strings.Split(rec, "\x00")
	if len(f) < headerFields {
		return Commit{}, fmt.Errorf("gitx: malformed log record %q", rec)
	}
	date, err := time.Parse(time.RFC3339, f[4])
	if err != nil {
		return Commit{}, fmt.Errorf("gitx: bad commit date %q: %w", f[4], err)
	}
	c := Commit{
		SHA: f[0], Short: f[1], Author: f[2], Email: f[3], Date: date,
		Parents: strings.Fields(f[5]),
		Subject: f[6],
		Body:    strings.TrimSpace(f[7]),
	}
	// Remaining tokens: status, path pairs (a leading newline may precede the first).
	var toks []string
	for _, t := range f[headerFields:] {
		if len(toks) == 0 {
			t = strings.TrimLeft(t, "\n")
		}
		if t != "" {
			toks = append(toks, t)
		}
	}
	if len(toks)%2 != 0 {
		return Commit{}, fmt.Errorf("gitx: malformed name-status in commit %s: %q", c.SHA, toks)
	}
	for i := 0; i < len(toks); i += 2 {
		if len(toks[i]) != 1 {
			return Commit{}, fmt.Errorf("gitx: unexpected status %q in commit %s", toks[i], c.SHA)
		}
		c.Files = append(c.Files, FileChange{Status: toks[i][0], Path: toks[i+1]})
	}
	return c, nil
}

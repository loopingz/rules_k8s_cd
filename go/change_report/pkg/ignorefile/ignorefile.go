// Package ignorefile reads the ignore file of change_report and the release
// tool (.release-ignore by default) and the Release-Ignore commit trailer.
//
// Each line of the file is either a commit, a full SHA followed by the names
// it is ignored for (release components, target labels; none: all of them),
// or a glob of files never attributed. `#` starts a comment:
//
//	# reformat
//	0123456789abcdef0123456789abcdef01234567
//	# a dependency pin that only concerns the UI
//	89abcdef0123456789abcdef0123456789abcdef api cli //tools:gen
//	docs/**
//
// A commit message trailer `Release-Ignore: api, cli` (`*` or empty: all)
// ignores its commit the same way.
package ignorefile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DefaultName is the file read at the workspace root unless told otherwise.
const DefaultName = ".release-ignore"

// File is a parsed ignore file.
type File struct {
	// Name is the file path, for messages.
	Name string
	// Commits maps a full SHA to the names it is ignored for; an empty
	// list means every name.
	Commits map[string][]string
	// Paths are globs of files never attributed.
	Paths []string
}

var (
	fullSHA  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	shortSHA = regexp.MustCompile(`^[0-9a-f]{7,39}$`)
	trailer  = regexp.MustCompile(`(?im)^release-ignore:[ \t]*(.*?)[ \t]*$`)
)

// Parse reads an ignore file. Malformed lines are skipped with a warning.
func Parse(name string, data []byte) (File, []string) {
	f := File{Name: name, Commits: map[string][]string{}}
	var warns []string
	for i, line := range strings.Split(string(data), "\n") {
		if j := strings.IndexByte(line, '#'); j >= 0 {
			line = line[:j]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		at := fmt.Sprintf("%s:%d", name, i+1)
		switch first := strings.ToLower(fields[0]); {
		case fullSHA.MatchString(first):
			f.Commits[first] = append(f.Commits[first], fields[1:]...)
		case shortSHA.MatchString(first):
			warns = append(warns, fmt.Sprintf("%s: commit %s is not a full SHA; skipped", at, fields[0]))
		case len(fields) > 1:
			warns = append(warns, fmt.Sprintf("%s: %q is neither a full commit SHA nor a single path glob; skipped", at, strings.TrimSpace(line)))
		default:
			f.Paths = append(f.Paths, fields[0])
		}
	}
	return f, warns
}

// Load reads dir/name; a missing file is an empty File.
func Load(dir, name string) (File, []string, error) {
	p := name
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, name)
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return File{Name: name, Commits: map[string][]string{}}, nil, nil
	}
	if err != nil {
		return File{}, nil, err
	}
	f, warns := Parse(name, data)
	return f, warns, nil
}

// Trailer returns the names of the Release-Ignore trailers of a commit
// message; all is true when one has no name or `*`.
func Trailer(body string) (names []string, all, found bool) {
	for _, m := range trailer.FindAllStringSubmatch(body, -1) {
		found = true
		for _, n := range strings.FieldsFunc(m[1], func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
			if n == "*" {
				all = true
			} else {
				names = append(names, n)
			}
		}
		if strings.TrimSpace(m[1]) == "" {
			all = true
		}
	}
	return names, all, found
}

// Ignores reports whether the commit sha (message body) is ignored for any of
// names, by the file or by a trailer.
func (f File) Ignores(sha, body string, names ...string) bool {
	if scopes, ok := f.Commits[sha]; ok && (len(scopes) == 0 || anyIn(scopes, names)) {
		return true
	}
	tn, all, found := Trailer(body)
	return found && (all || anyIn(tn, names))
}

func anyIn(scopes, names []string) bool {
	for _, s := range scopes {
		for _, n := range names {
			if s == n {
				return true
			}
		}
	}
	return false
}

// IsLabel tells a target label from a release component name.
func IsLabel(name string) bool {
	return strings.HasPrefix(name, "//") || strings.HasPrefix(name, "@") || strings.HasPrefix(name, ":")
}

// Check warns about the file's commits that are not in the repository
// (exists) and about the names known rejects; only names of the kind the
// caller knows are checked (labels or component names, see IsLabel).
func (f File) Check(exists func(sha string) bool, labels bool, known func(name string) bool) []string {
	var warns []string
	for sha, scopes := range f.Commits {
		if exists != nil && !exists(sha) {
			warns = append(warns, fmt.Sprintf("%s: commit %s is not in the repository; ignored", f.Name, sha))
		}
		for _, s := range scopes {
			if IsLabel(s) == labels && !known(s) {
				kind := "release component"
				if labels {
					kind = "target"
				}
				warns = append(warns, fmt.Sprintf("%s: commit %s: no %s %s", f.Name, sha, kind, s))
			}
		}
	}
	return warns
}

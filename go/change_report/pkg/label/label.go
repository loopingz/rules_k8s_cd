// Package label parses and formats Bazel labels.
package label

import (
	"fmt"
	"strings"
)

// Label is a parsed Bazel label. Repo is empty for labels in the main repository.
type Label struct {
	Repo string
	Pkg  string
	Name string
}

// Parse parses a Bazel label such as "@@//pkg:name", "//pkg/sub" (name defaults
// to the last path segment), "@@repo+//x:y" or "@repo//x:y".
func Parse(s string) (Label, error) {
	var l Label
	var rest string
	switch {
	case strings.HasPrefix(s, "@@"):
		rest = s[2:]
	case strings.HasPrefix(s, "@"):
		rest = s[1:]
	case strings.HasPrefix(s, "//"):
		rest = s
	default:
		return Label{}, fmt.Errorf("label %q: not a Bazel label", s)
	}

	idx := strings.Index(rest, "//")
	if idx < 0 {
		return Label{}, fmt.Errorf("label %q: missing \"//\"", s)
	}
	if strings.HasPrefix(s, "@") {
		l.Repo = rest[:idx]
	}
	body := rest[idx+2:]

	if colon := strings.Index(body, ":"); colon >= 0 {
		l.Pkg = body[:colon]
		l.Name = body[colon+1:]
	} else {
		l.Pkg = body
		if i := strings.LastIndex(body, "/"); i >= 0 {
			l.Name = body[i+1:]
		} else {
			l.Name = body
		}
	}

	if l.Name == "" {
		return Label{}, fmt.Errorf("label %q: empty target name", s)
	}
	return l, nil
}

// String returns "//pkg:name" for main-repo labels and "@@repo//pkg:name" otherwise.
func (l Label) String() string {
	if l.IsMain() {
		return fmt.Sprintf("//%s:%s", l.Pkg, l.Name)
	}
	return fmt.Sprintf("@@%s//%s:%s", l.Repo, l.Pkg, l.Name)
}

// IsMain reports whether the label belongs to the main repository.
func (l Label) IsMain() bool {
	return l.Repo == ""
}

// Path returns the workspace-relative file path of a main-repo label,
// e.g. "//a/b:c" -> "a/b/c" and "//:BUILD" -> "BUILD".
func (l Label) Path() string {
	if l.Pkg == "" {
		return l.Name
	}
	return l.Pkg + "/" + l.Name
}

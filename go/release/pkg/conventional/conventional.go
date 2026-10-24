// Package conventional parses Conventional Commit messages and derives
// version bumps from them.
package conventional

import (
	"fmt"
	"regexp"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

// Parsed is the conventional-commit view of a commit. Non-conventional
// commits (including git's own `Merge ...` and `Revert "..."` subjects) have
// Conventional false, an empty Type and the whole subject as Description.
type Parsed struct {
	Type, Scope, Description string
	Breaking                 bool
	// ReleaseAs is the version forced by a valid Release-As footer (the
	// greatest one when several are present).
	ReleaseAs    *semver.Version
	Conventional bool
	// Warnings reports malformed footers that were ignored.
	Warnings []string
}

var (
	subjectRe   = regexp.MustCompile(`^(\w+)(\(([^)]*)\))?(!)?: (.+)$`)
	releaseAsRe = regexp.MustCompile(`(?i)^release-as:\s*(.*?)\s*$`)
)

// Parse parses the subject and footers of c.
func Parse(c gitx.Commit) Parsed {
	var p Parsed
	if m := subjectRe.FindStringSubmatch(c.Subject); m != nil {
		p.Type, p.Scope, p.Description = m[1], m[3], m[5]
		p.Breaking = m[4] == "!"
		p.Conventional = true
	} else {
		p.Description = c.Subject
	}
	for _, line := range strings.Split(c.Body, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "BREAKING CHANGE:") || strings.HasPrefix(line, "BREAKING-CHANGE:") {
			p.Breaking = true
		}
		if m := releaseAsRe.FindStringSubmatch(line); m != nil {
			ver, err := semver.Parse(m[1])
			if err != nil {
				p.Warnings = append(p.Warnings, fmt.Sprintf("ignoring invalid Release-As %q in commit %s", m[1], c.Short))
				continue
			}
			if p.ReleaseAs == nil || p.ReleaseAs.Less(ver) {
				p.ReleaseAs = &ver
			}
		}
	}
	return p
}

// BumpFor returns the bump implied by p at version current: breaking is major
// (minor below 1.0.0), feat is minor (patch below 1.0.0), fix, perf, revert,
// unknown types and non-conventional commits are patch; only chore, docs,
// style, test, ci, build and refactor do not release.
func BumpFor(p Parsed, current semver.Version) semver.Bump {
	pre1 := current.Major < 1
	switch {
	case p.Breaking:
		if pre1 {
			return semver.Minor
		}
		return semver.Major
	case !p.Conventional:
		return semver.Patch
	}
	switch strings.ToLower(p.Type) {
	case "feat":
		if pre1 {
			return semver.Patch
		}
		return semver.Minor
	case "chore", "docs", "style", "test", "ci", "build", "refactor":
		return semver.None
	}
	// fix, perf, revert and unknown types: never miss a release.
	return semver.Patch
}

// Next computes the next version. With a nil current the first release is
// initial, provided a commit is releasable. A Release-As footer (greatest
// valid) wins when greater than current, otherwise it is ignored with a
// warning.
func Next(current *semver.Version, initial semver.Version, commits []Parsed) (semver.Version, bool, []string) {
	var warnings []string
	var forced *semver.Version
	for _, p := range commits {
		warnings = append(warnings, p.Warnings...)
		if p.ReleaseAs == nil {
			continue
		}
		if current != nil && !current.Less(*p.ReleaseAs) {
			warnings = append(warnings, fmt.Sprintf("ignoring Release-As %s: not greater than current %s", p.ReleaseAs, current))
			continue
		}
		if forced == nil || forced.Less(*p.ReleaseAs) {
			r := *p.ReleaseAs
			forced = &r
		}
	}
	if forced != nil {
		return *forced, true, warnings
	}

	ref := initial
	if current != nil {
		ref = *current
	}
	best := semver.None
	for _, p := range commits {
		if b := BumpFor(p, ref); b > best {
			best = b
		}
	}
	if best == semver.None {
		return semver.Version{}, false, warnings
	}
	if current == nil {
		return initial, true, warnings
	}
	return current.Bump(best), true, warnings
}

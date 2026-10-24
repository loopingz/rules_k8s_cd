// Package semver implements the strict MAJOR.MINOR.PATCH versions used by
// release VERSION files (no pre-release or build metadata).
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a MAJOR.MINOR.PATCH version.
type Version struct {
	Major, Minor, Patch int
}

// Bump is the kind of version increment.
type Bump int

// Bump kinds, ordered by increasing significance.
const (
	None Bump = iota
	Patch
	Minor
	Major
)

// Parse parses "X.Y.Z" with an optional leading "v". Pre-release and build
// suffixes, missing parts and leading zeros are rejected.
func Parse(s string) (Version, error) {
	orig := s
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q: want MAJOR.MINOR.PATCH", orig)
	}
	var n [3]int
	for i, p := range parts {
		if p == "" || (len(p) > 1 && p[0] == '0') {
			return Version{}, fmt.Errorf("invalid version %q: bad component %q", orig, p)
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return Version{}, fmt.Errorf("invalid version %q: bad component %q", orig, p)
			}
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q: %v", orig, err)
		}
		n[i] = v
	}
	return Version{n[0], n[1], n[2]}, nil
}

// String returns "X.Y.Z".
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Less reports whether v sorts before o.
func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	return v.Patch < o.Patch
}

// Bump mechanically increments v by kind (resetting lower components). The
// pre-1.0 downgrade rules are the caller's concern.
func (v Version) Bump(kind Bump) Version {
	switch kind {
	case Major:
		return Version{v.Major + 1, 0, 0}
	case Minor:
		return Version{v.Major, v.Minor + 1, 0}
	case Patch:
		return Version{v.Major, v.Minor, v.Patch + 1}
	}
	return v
}

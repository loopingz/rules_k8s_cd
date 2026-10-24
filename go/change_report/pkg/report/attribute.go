package report

import (
	"path"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/extdeps"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/graph"
)

const (
	moduleFile = "MODULE.bazel"
	lockFile   = "MODULE.bazel.lock"
)

// depGlobs are dependency-looking files. Without a Mapping.FileRepos entry
// they are global, since we cannot tell which repos they feed.
var depGlobs = [...]string{
	"**/go.mod", "**/go.sum", "**/package.json", "**/pnpm-lock.yaml", "**/package-lock.json",
	"**/yarn.lock", "**/requirements*.txt", "**/*.lock", "**/Chart.yaml", "**/Chart.lock",
}

// classified is a non-merge commit whose files have been sorted into global
// files and repo-scoped files.
type classified struct {
	c      gitx.Commit
	global map[string]bool
	// scoped maps a file to groups of repo patterns: the file is attributed to
	// a target with a closure repo that matches some pattern of every group.
	scoped map[string][][]string
}

// classify applies the global and external-dependency rules to c's files.
func classify(in Input, c gitx.Commit) classified {
	cc := classified{c: c, global: map[string]bool{}, scoped: map[string][][]string{}}

	touched := map[string]bool{}
	for _, f := range c.Files {
		touched[f.Path] = true
	}
	// Level 2: the lockfile diff names the repos that actually changed.
	var changed []string
	level2 := false
	if touched[lockFile] && len(c.Parents) == 1 && in.Tracked != nil && in.Tracked(lockFile) && in.FileAt != nil {
		before, okb := in.FileAt(c.SHA+"^", lockFile)
		after, oka := in.FileAt(c.SHA, lockFile)
		if okb && oka {
			changed, level2 = extdeps.ChangedRepos(before, after)
		}
	}

	for _, f := range c.Files {
		p := f.Path
		switch {
		case p == lockFile:
			// Level 2 ok but naming no repo leaves the change unexplained.
			if level2 && len(changed) > 0 {
				cc.scoped[p] = [][]string{changed}
			} else {
				cc.global[p] = true
			}
		case p == moduleFile:
			if level2 && len(changed) > 0 && moduleExplained(in, c.SHA) {
				cc.scoped[p] = [][]string{changed}
			} else {
				cc.global[p] = true
			}
		case isGlobal(in, p):
			cc.global[p] = true
		case len(in.Mapping.FileRepos[p]) > 0:
			mapped := in.Mapping.FileRepos[p]
			groups := [][]string{mapped}
			if level2 {
				// Narrow to the changed repos, but never to nothing: the file
				// did change, so fall back to the mapped set.
				var both []string
				for _, ch := range changed {
					for _, m := range mapped {
						if overlap(m, ch) {
							both = append(both, ch)
							break
						}
					}
				}
				if len(both) > 0 {
					groups = append(groups, both)
				}
			}
			cc.scoped[p] = groups
		case isDependencyFile(p):
			cc.global[p] = true
		}
	}
	return cc
}

// moduleExplained reports whether MODULE.bazel's change in sha only touched
// bazel_dep calls. Missing content on either side is not explained.
func moduleExplained(in Input, sha string) bool {
	before, okb := in.FileAt(sha+"^", moduleFile)
	after, oka := in.FileAt(sha, moduleFile)
	return okb && oka && extdeps.ModuleChangeExplained(before, after)
}

// attributed returns the files of the commit that affect a target with the
// given closure.
func (cc classified) attributed(cl graph.Closure) []string {
	var out []string
	for _, f := range cc.c.Files {
		if cc.hits(cl, f) {
			out = append(out, f.Path)
		}
	}
	return out
}

func (cc classified) hits(cl graph.Closure, f gitx.FileChange) bool {
	p := f.Path
	if cc.global[p] {
		return true
	}
	if groups, ok := cc.scoped[p]; ok && reposMatch(cl.Repos, groups) {
		return true
	}
	// Exact match, or p is under a source directory used as a label
	// (srcs = ["chart"]).
	for q := p; q != "."; q = path.Dir(q) {
		if i := sort.SearchStrings(cl.Files, q); i < len(cl.Files) && cl.Files[i] == q {
			return true
		}
		if !strings.Contains(q, "/") {
			break
		}
	}
	// An added or deleted file can change a glob in its package.
	if f.Status == 'A' || f.Status == 'D' {
		dir := path.Dir(p)
		if dir == "." {
			dir = ""
		}
		for _, pkg := range cl.Packages {
			if under(dir, pkg) {
				return true
			}
		}
	}
	return false
}

// reposMatch reports whether some repo matches a pattern of every group.
func reposMatch(repos []string, groups [][]string) bool {
	for _, r := range repos {
		all := true
		for _, g := range groups {
			if !anyMatch(g, r) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

func anyMatch(patterns []string, repo string) bool {
	for _, p := range patterns {
		if extdeps.MatchRepo(p, repo) {
			return true
		}
	}
	return false
}

func isPattern(s string) bool { return strings.HasPrefix(s, "*") || strings.HasSuffix(s, "*") }

// overlap reports whether two repo patterns may match a common repo. Two
// wildcard patterns are conservatively assumed to overlap.
func overlap(a, b string) bool {
	switch {
	case !isPattern(a) && !isPattern(b):
		return a == b
	case !isPattern(a):
		return extdeps.MatchRepo(b, a)
	case !isPattern(b):
		return extdeps.MatchRepo(a, b)
	}
	return true
}

// under reports whether dir is pkg or below it; the root package "" contains
// everything.
func under(dir, pkg string) bool {
	return pkg == "" || dir == pkg || strings.HasPrefix(dir, pkg+"/")
}

// isGlobal reports whether p affects every target (MODULE.bazel and the
// lockfile are handled by the caller).
func isGlobal(in Input, p string) bool {
	b := path.Base(p)
	switch {
	case p == "WORKSPACE", p == "WORKSPACE.bazel", p == "WORKSPACE.bzlmod",
		p == ".bazelrc", p == ".bazelversion", p == ".bazelignore", b == "REPO.bazel",
		strings.HasSuffix(b, ".bazelrc"), strings.HasSuffix(b, ".MODULE.bazel"):
		return true
	}
	for _, g := range in.GlobalGlobs {
		if MatchGlob(g, p) {
			return true
		}
	}
	for _, m := range in.ModuleBzlFiles {
		if m == p {
			return true
		}
	}
	for _, lp := range in.Mapping.LocalPaths {
		lp = path.Clean(lp)
		if lp == ".." || strings.HasPrefix(lp, "../") || strings.HasPrefix(lp, "/") {
			continue // outside the workspace
		}
		if lp == "." {
			lp = ""
		}
		if under(p, lp) {
			return true
		}
	}
	return false
}

func isDependencyFile(p string) bool {
	for _, g := range depGlobs {
		if MatchGlob(g, p) {
			return true
		}
	}
	return false
}

// MatchGlob matches a slash-separated path against pattern. A "**" segment
// matches zero or more segments; other segments use path.Match, so "*" stays
// within a segment. Malformed patterns match nothing.
func MatchGlob(pattern, p string) bool {
	return matchSegs(strings.Split(pattern, "/"), strings.Split(p, "/"))
}

func matchSegs(ps, ss []string) bool {
	if len(ps) == 0 {
		return len(ss) == 0
	}
	if ps[0] == "**" {
		return matchSegs(ps[1:], ss) || (len(ss) > 0 && matchSegs(ps, ss[1:]))
	}
	if len(ss) == 0 {
		return false
	}
	ok, err := path.Match(ps[0], ss[0])
	return err == nil && ok && matchSegs(ps[1:], ss[1:])
}

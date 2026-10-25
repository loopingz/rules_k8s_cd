package report

import (
	"fmt"
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
// they are global when the mapping records no file at all, since we cannot
// tell which repos they feed. Otherwise the mapping lists every file a module
// extension reads (the lockfile's recorded inputs, plus the files named in
// the root module's tags): an unlisted file feeds no repository, and only
// affects the targets whose closure contains it.
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
	// owner maps an added or deleted file to the package owning it, when
	// Input.Packages is known.
	owner map[string]string
}

// packageOwner returns the deepest package of pkgs containing the file p.
func packageOwner(pkgs map[string]bool, p string) (string, bool) {
	for dir := path.Dir(p); ; dir = path.Dir(dir) {
		if dir == "." || dir == "/" {
			dir = ""
		}
		if pkgs[dir] {
			return dir, true
		}
		if dir == "" {
			return "", false
		}
	}
}

// mappedUsage tells whether a mapped dependency file's repo patterns match
// some repo of the universe closures, memoized per file.
type mappedUsage struct {
	repos []string // union of the universe closures' repos
	used  map[string]bool
}

func newMappedUsage(in Input) *mappedUsage {
	set := map[string]bool{}
	for _, l := range in.Universe {
		for _, r := range in.Closures[l].Repos {
			set[r] = true
		}
	}
	return &mappedUsage{repos: sortedKeys(set), used: map[string]bool{}}
}

// isUsed reports whether some pattern of patterns matches a closure repo;
// first is true the first time p is asked about.
func (u *mappedUsage) isUsed(p string, patterns []string) (used, first bool) {
	if v, ok := u.used[p]; ok {
		return v, false
	}
	v := false
	for _, r := range u.repos {
		if anyMatch(patterns, r) {
			v = true
			break
		}
	}
	u.used[p] = v
	return v, true
}

// classify applies the global and external-dependency rules to c's files.
// Warnings are appended to warns.
func classify(in Input, pkgs map[string]bool, c gitx.Commit, usage *mappedUsage, warns *[]string) classified {
	cc := classified{c: c, global: map[string]bool{}, scoped: map[string][][]string{}, owner: map[string]string{}}
	if pkgs != nil {
		for _, f := range c.Files {
			if f.Status == 'A' || f.Status == 'D' {
				if o, ok := packageOwner(pkgs, f.Path); ok {
					cc.owner[f.Path] = o
				}
			}
		}
	}

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
			if used, first := usage.isUsed(p, mapped); !used {
				// The mapping names repos no target uses (e.g. other
				// canonical-name conventions): it cannot be trusted.
				if first {
					*warns = append(*warns, fmt.Sprintf("dependency file %s maps to repositories not used by any target; treating it as global", p))
				}
				cc.global[p] = true
				continue
			}
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
		case isDependencyFile(p) && len(in.Mapping.FileRepos) == 0:
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
// Paths for which ignore returns true are never attributed (ignore may be nil).
func (cc classified) attributed(cl graph.Closure, ignore func(string) bool) []string {
	var out []string
	for _, f := range cc.c.Files {
		if ignore != nil && ignore(f.Path) {
			continue
		}
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
		if o, ok := cc.owner[p]; ok {
			i := sort.SearchStrings(cl.Packages, o)
			return i < len(cl.Packages) && cl.Packages[i] == o
		}
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

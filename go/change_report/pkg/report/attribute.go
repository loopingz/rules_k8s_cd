package report

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/extdeps"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/graph"
	"loopingz.com/rules_k8s_cd/change_report/pkg/stmt"
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
	// owner maps a deleted file to the package owning it, when Input.Packages
	// is known.
	owner map[string]string
	// under maps a file to directories: it affects the targets with a closure
	// package in one of them (.bazelignore entries).
	under map[string][]string
	// calls maps an edited BUILD file to the names of the top-level calls the
	// edit changed, nil when it may change any target of the package (a load,
	// an assignment, an unnamed call, or a file added or deleted).
	calls map[string]map[string]bool
	// globs tells, per package, whether a deleted file may change a target of
	// a closure (see deletedHits); filled lazily, shared by all commits.
	globs *globCache
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

// sameConvention reports whether patterns use the canonical-name separator
// of the universe closures ("~" for Bazel 7, "+" since Bazel 8); true when
// the closures use no external repo.
func (u *mappedUsage) sameConvention(patterns []string) bool {
	tilde := func(names []string) (yes, any bool) {
		for _, n := range names {
			if strings.Contains(n, "~") {
				return true, true
			}
			if strings.Contains(n, "+") {
				any = true
			}
		}
		return false, any
	}
	ut, uany := tilde(u.repos)
	pt, pany := tilde(patterns)
	return !uany || !pany || ut == pt
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
func classify(in Input, pkgs map[string]bool, globs *globCache, c gitx.Commit, usage *mappedUsage, warns *[]string) classified {
	cc := classified{c: c, global: map[string]bool{}, scoped: map[string][][]string{}, owner: map[string]string{},
		under: map[string][]string{}, calls: map[string]map[string]bool{}, globs: globs}
	for _, f := range c.Files {
		if isBuildFile(f.Path) && f.Status == 'M' {
			cc.calls[f.Path] = changedCalls(in, c.SHA, f.Path)
		}
	}
	if pkgs != nil {
		for _, f := range c.Files {
			if f.Status == 'D' {
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
			} else if pats, ok := moduleDepScope(in, c, touched[lockFile], level2, changed); ok {
				cc.scoped[p] = [][]string{pats}
			} else if pats, ok := moduleScope(in, c.SHA, usage); ok {
				cc.scoped[p] = [][]string{pats}
			} else {
				cc.global[p] = true
			}
		case p == bazelIgnore && !isGlobalGlob(in, p):
			if dirs, ok := ignoreScope(in, c.SHA); ok {
				cc.under[p] = dirs
			} else {
				cc.global[p] = true
			}
		case isGlobal(in, p):
			cc.global[p] = true
		case len(in.Mapping.FileRepos[p]) > 0:
			mapped := in.Mapping.FileRepos[p]
			if used, first := usage.isUsed(p, mapped); !used && !usage.sameConvention(mapped) {
				// The mapping names repos no target uses in another
				// canonical-name convention (Bazel 7 "~"): it cannot be
				// trusted. In the closures' convention, they are just
				// repos no target of the universe uses.
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

const bazelIgnore = ".bazelignore"

func isBuildFile(p string) bool {
	b := path.Base(p)
	return b == "BUILD" || b == "BUILD.bazel"
}

// changedCalls returns the names of the top-level calls sha changed in the
// BUILD file p, nil when it may have changed any target of the package.
func changedCalls(in Input, sha, p string) map[string]bool {
	if in.FileAt == nil {
		return nil
	}
	before, okb := in.FileAt(sha+"^", p)
	after, oka := in.FileAt(sha, p)
	if !okb || !oka {
		return nil
	}
	names := map[string]bool{}
	for _, s := range stmt.Changed(stmt.Split(string(before)), stmt.Split(string(after))) {
		n := s.Kwarg("name")
		if s.Call == "" || s.Assign != "" || n == "" || s.Call == "load" || s.Call == "package" {
			return nil
		}
		names[n] = true
	}
	return names
}

// globCache tells whether a deleted file of a package may change one of a
// closure's targets: when one of its calls in the package's BUILD file at
// HEAD uses glob(), comes from a macro (which may glob inside), or when the
// file cannot be read or has a glob() outside any named call.
type globCache struct {
	in   Input
	head map[string][]stmt.Stmt // package -> statements, nil: unreadable
}

func newGlobCache(in Input) *globCache {
	return &globCache{in: in, head: map[string][]stmt.Stmt{}}
}

func (g *globCache) mayGlob(pkg string, cl graph.Closure) bool {
	if g == nil || g.in.FileAt == nil {
		return true
	}
	stmts, seen := g.head[pkg]
	if !seen {
		stmts = nil
		for _, b := range []string{"BUILD.bazel", "BUILD"} {
			if src, ok := g.in.FileAt(g.in.Head, path.Join(pkg, b)); ok {
				stmts = stmt.Split(string(src))
				break
			}
		}
		g.head[pkg] = stmts
	}
	if stmts == nil {
		return true
	}
	for _, m := range cl.Macros {
		if strings.HasPrefix(m, pkg+":") {
			return true
		}
	}
	for _, s := range stmts {
		if !strings.Contains(s.Text, "glob(") {
			continue
		}
		n := s.Kwarg("name")
		if n == "" || s.Assign != "" {
			return true
		}
		if i := sort.SearchStrings(cl.Calls, pkg+":"+n); i < len(cl.Calls) && cl.Calls[i] == pkg+":"+n {
			return true
		}
	}
	return false
}

func isGlobalGlob(in Input, p string) bool {
	for _, g := range in.GlobalGlobs {
		if MatchGlob(g, p) {
			return true
		}
	}
	return false
}

// ignoreScope returns the directories added to or removed from .bazelignore
// in sha: only packages under them appear or disappear. ok is false when
// either side is unavailable or an entry is not a plain directory.
func ignoreScope(in Input, sha string) (dirs []string, ok bool) {
	if in.FileAt == nil {
		return nil, false
	}
	before, okb := in.FileAt(sha+"^", bazelIgnore)
	after, oka := in.FileAt(sha, bazelIgnore)
	if !okb && !oka {
		return nil, false
	}
	entries := func(b []byte) map[string]bool {
		m := map[string]bool{}
		for _, l := range strings.Split(string(b), "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
				m[strings.TrimSuffix(path.Clean(l), "/")] = true
			}
		}
		return m
	}
	eb, ea := entries(before), entries(after)
	for _, pair := range [][2]map[string]bool{{eb, ea}, {ea, eb}} {
		for e := range pair[0] {
			if pair[1][e] {
				continue
			}
			if e == "." || strings.HasPrefix(e, "..") || strings.ContainsAny(e, "*?[") {
				return nil, false
			}
			dirs = append(dirs, e)
		}
	}
	sort.Strings(dirs)
	return dirs, true
}

// moduleDepScope scopes a MODULE.bazel edit of bazel_dep and override calls
// (extdeps.ModuleDepChanges) to the repos of those modules plus the repos the
// lockfile diff names. It needs the lockfile at the commit and its parent: an
// unchanged lockfile means no registry module changed version.
func moduleDepScope(in Input, c gitx.Commit, lockTouched, level2 bool, changed []string) (patterns []string, ok bool) {
	if in.FileAt == nil || len(c.Parents) != 1 {
		return nil, false
	}
	before, okb := in.FileAt(c.SHA+"^", moduleFile)
	after, oka := in.FileAt(c.SHA, moduleFile)
	if !okb || !oka {
		return nil, false
	}
	if _, ok := in.FileAt(c.SHA+"^", lockFile); !ok {
		return nil, false
	}
	if _, ok := in.FileAt(c.SHA, lockFile); !ok {
		return nil, false
	}
	if lockTouched && (!level2 || len(changed) == 0) {
		return nil, false // the lockfile diff could not be read, or explains nothing
	}
	mods, ok := extdeps.ModuleDepChanges(before, after)
	if !ok || len(mods) == 0 {
		return nil, false
	}
	for _, m := range mods {
		patterns = append(patterns, m+"+", m+"++*")
	}
	return append(patterns, changed...), true
}

// moduleScope returns the repo patterns MODULE.bazel's change in sha can
// affect, statement by statement (extdeps.ModuleEditScope). ok is false when
// it cannot tell, or when a pattern matches no repo of the universe closures:
// such repos may be toolchains, which target closures do not show.
func moduleScope(in Input, sha string, usage *mappedUsage) (patterns []string, ok bool) {
	if in.FileAt == nil {
		return nil, false
	}
	before, okb := in.FileAt(sha+"^", moduleFile)
	after, oka := in.FileAt(sha, moduleFile)
	if !okb || !oka {
		return nil, false
	}
	patterns, ok = extdeps.ModuleEditScope(before, after, extdeps.MainRepoNames(after))
	if !ok {
		return nil, false
	}
	for _, p := range patterns {
		// A use_repo_rule repo ("*+name") that no target uses changes none
		// (ModuleEditScope keeps those registered as toolchains global); an
		// extension's repos may be toolchains, which closures do not show.
		if strings.HasPrefix(p, "*+") {
			continue
		}
		if used, _ := usage.isUsed(moduleFile+"\x00"+p, []string{p}); !used {
			return nil, false
		}
	}
	// nothing but formatting or comments changed: no repo
	if len(patterns) == 0 {
		return []string{}, true
	}
	return patterns, true
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
	if dirs, ok := cc.under[p]; ok {
		for _, pkg := range cl.Packages {
			for _, d := range dirs {
				if under(pkg, d) {
					return true
				}
			}
		}
		return false
	}
	// An edited BUILD file of a closure package: only the calls it changed.
	if names, ok := cc.calls[p]; ok && names != nil {
		pkg := path.Dir(p)
		if pkg == "." {
			pkg = ""
		}
		if i := sort.SearchStrings(cl.Files, p); i < len(cl.Files) && cl.Files[i] == p {
			for n := range names {
				if i := sort.SearchStrings(cl.Calls, pkg+":"+n); i < len(cl.Calls) && cl.Calls[i] == pkg+":"+n {
					return true
				}
			}
		}
		return false
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
	// A deleted file can change a glob in its package. An added one is in the
	// closure computed at HEAD when a glob matches it (exact match above),
	// unless it was deleted again, which changes nothing.
	if f.Status == 'D' {
		if o, ok := cc.owner[p]; ok {
			i := sort.SearchStrings(cl.Packages, o)
			return i < len(cl.Packages) && cl.Packages[i] == o && cc.globs.mayGlob(o, cl)
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
		p == ".bazelrc", p == ".bazelversion", p == bazelIgnore, b == "REPO.bazel",
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

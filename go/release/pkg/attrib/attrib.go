// Package attrib decides, per release component, which commits since the
// component's baseline affect it through its Bazel dependency graph.
//
// Each component becomes one synthetic change_report target, labelled with
// the component's own label, whose closure is the union of the closures of
// the component's targets. Release commits, and every component's version
// file and changelog, are never attributed: a release can never trigger
// another release.
package attrib

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/graph"
	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

// releaseSubject matches the subject of a release commit,
// "chore(release): <name> <version>", optionally followed by the " (#N)"
// suffix forges add when squash-merging a pull request.
var releaseSubject = regexp.MustCompile(`^chore\(release\): (\S+) (\d+\.\d+\.\d+)(?: \(#\d+\))?$`)

// ParseReleaseSubject returns the component name and version of a release
// commit subject; ok is false if subject is not one.
func ParseReleaseSubject(subject string) (name string, version semver.Version, ok bool) {
	m := releaseSubject.FindStringSubmatch(subject)
	if m == nil {
		return "", semver.Version{}, false
	}
	v, err := semver.Parse(m[2])
	if err != nil {
		return "", semver.Version{}, false
	}
	return m[1], v, true
}

// IsReleaseCommit reports whether c is a release commit.
func IsReleaseCommit(c gitx.Commit) bool {
	_, _, ok := ParseReleaseSubject(c.Subject)
	return ok
}

// Result is the attribution of one component.
type Result struct {
	Component component.Component
	// Targets are the universe labels belonging to the component, sorted.
	Targets []string
	// Commits attributed to the component, newest first.
	Commits []gitx.Commit
	// Files maps an attributed commit SHA to the files that caused its
	// attribution, sorted (report.Target.CommitFiles).
	Files map[string][]string
}

// Attribute attributes the commits up to head to comps, keyed by component
// label. states must hold a state for every component; its BaselineSHA
// (possibly report.RootBaseline) is where attribution starts. o.Universe is
// set by Attribute. Every warning is returned and, if o.Warn is non-nil, also
// passed to it. globalGlobs are files affecting every component.
//
// Release commits of comps are dropped whole, and every component's version file and
// changelog are never attributed. Bump files are attributed like any file: a
// non-release change to one is a real change of its owner.
func Attribute(o cache.Options, repo gitx.Repo, head string, comps []component.Component, states map[string]component.State, globalGlobs []string) (map[string]Result, []string, error) {
	var warnings []string
	callerWarn := o.Warn
	warn := func(s string) {
		warnings = append(warnings, s)
		if callerWarn != nil {
			callerWarn(s)
		}
	}

	perComp := map[string]string{}
	var labels []string
	exprs := make([]string, 0, len(comps))
	for _, c := range comps {
		st, ok := states[c.Label]
		if !ok || st.BaselineSHA == "" {
			return nil, warnings, fmt.Errorf("component %s has no baseline", c.Label)
		}
		perComp[c.Label] = st.BaselineSHA
		labels = append(labels, c.Label)
		exprs = append(exprs, c.UniverseExpr())
	}
	if len(comps) == 0 {
		return map[string]Result{}, nil, nil
	}

	o.Universe = strings.Join(exprs, " + ")
	o.Warn = warn
	snap, _, err := cache.Load(o)
	if err != nil {
		return nil, warnings, err
	}
	inUniverse := make(map[string]bool, len(snap.Universe))
	for _, l := range snap.Universe {
		inUniverse[l] = true
	}

	results := make(map[string]Result, len(comps))
	closures := make(map[string]graph.Closure, len(comps))
	byLabel := make(map[string]component.Component, len(comps))
	for _, c := range comps {
		targets, err := members(o.Runner, c, inUniverse)
		if err != nil {
			return nil, warnings, err
		}
		if len(targets) == 0 {
			if c.TargetsQuery != "" {
				return nil, warnings, fmt.Errorf("component %s: targets_query %q matches no targets", c.Label, c.TargetsQuery)
			}
			return nil, warnings, fmt.Errorf("component %s has no targets", c.Label)
		}
		cls := make([]graph.Closure, len(targets))
		for i, t := range targets {
			cls[i] = snap.Closures[t]
		}
		closures[c.Label] = union(cls)
		byLabel[c.Label] = c
		results[c.Label] = Result{Component: c, Targets: targets}
	}

	baselines, bw := report.ResolveBaselines(repo, head, labels, "", perComp)
	for _, w := range bw {
		warn(w)
	}
	for _, l := range labels {
		if baselines[l] == "" {
			// Never miss a release: an unusable baseline means all history.
			baselines[l] = report.RootBaseline
			warn(fmt.Sprintf("baseline %s of %s is unusable; attributing all history", perComp[l], l))
		}
	}
	commits, inRange, err := report.CollectHistory(repo, head, baselines)
	if err != nil {
		return nil, warnings, err
	}

	// Version files and changelogs of every component are never attributed.
	releaseFiles := map[string]bool{}
	for _, c := range comps {
		if c.VersionFile != "" {
			releaseFiles[c.VersionFile] = true
		}
		if c.Changelog != "" {
			releaseFiles[c.Changelog] = true
		}
	}
	ignorePath := func(l, p string) bool {
		if releaseFiles[p] {
			return true
		}
		for _, g := range byLabel[l].Ignore {
			if report.MatchGlob(g, p) {
				return true
			}
		}
		return false
	}

	// Only a release commit of one of comps is dropped: "chore(release):
	// prepare 1.0.0" is an ordinary commit.
	names := make(map[string]bool, len(comps))
	for _, c := range comps {
		names[c.Name] = true
	}
	isRelease := func(c gitx.Commit) bool {
		n, _, ok := ParseReleaseSubject(c.Subject)
		return ok && names[n]
	}

	fileAt, tracked := report.GitAccessors(repo, warn)
	rep := report.Build(report.Input{
		Head:           head,
		Baselines:      baselines,
		Universe:       labels,
		Closures:       closures,
		Mapping:        snap.Mapping,
		Commits:        commits,
		InRange:        inRange,
		FileAt:         fileAt,
		Tracked:        tracked,
		GlobalGlobs:    globalGlobs,
		ModuleBzlFiles: snap.ModuleBzlFiles,
		IgnorePath:     ignorePath,
		IgnoreCommit:   isRelease,
	})
	for _, w := range rep.Warnings {
		warn(w)
	}

	for _, t := range rep.Targets {
		r, ok := results[t.Label]
		if !ok {
			continue
		}
		for _, sha := range t.Commits {
			r.Commits = append(r.Commits, rep.Commits[sha])
		}
		r.Files = t.CommitFiles
		results[t.Label] = r
	}
	return results, warnings, nil
}

// members returns the universe labels of c, sorted: its normalised targets,
// or the result of its targets_query intersected with the universe. A listed
// target missing from the universe is an error.
func members(r bazelq.Runner, c component.Component, inUniverse map[string]bool) ([]string, error) {
	raw := c.Targets
	if c.TargetsQuery != "" {
		var err error
		if raw, err = bazelq.Labels(r, "("+c.TargetsQuery+")"); err != nil {
			return nil, fmt.Errorf("component %s: targets_query: %w", c.Label, err)
		}
	}
	set := map[string]bool{}
	for _, s := range raw {
		l, err := label.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("component %s: %w", c.Label, err)
		}
		n := l.String()
		if !inUniverse[n] {
			if c.TargetsQuery != "" {
				continue
			}
			return nil, fmt.Errorf("component %s: target %s is not in the queried universe", c.Label, n)
		}
		set[n] = true
	}
	return sortedKeys(set), nil
}

// union merges closures, sorting and deduplicating each field.
func union(cls []graph.Closure) graph.Closure {
	files, pkgs, repos := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, c := range cls {
		for _, f := range c.Files {
			files[f] = true
		}
		for _, p := range c.Packages {
			pkgs[p] = true
		}
		for _, r := range c.Repos {
			repos[r] = true
		}
	}
	return graph.Closure{Files: sortedKeys(files), Packages: sortedKeys(pkgs), Repos: sortedKeys(repos)}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

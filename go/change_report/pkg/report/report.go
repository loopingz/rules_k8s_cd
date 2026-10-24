// Package report decides, for each target, whether it changed since its
// baseline commit and which commits caused it. It is pure: all repository
// access goes through the functions in Input.
//
// The guiding rule is that a changed target must never be reported as
// unchanged: every uncertainty resolves to "changed" / attributed.
package report

import (
	"fmt"
	"sort"

	"loopingz.com/rules_k8s_cd/change_report/pkg/extdeps"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitops"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/graph"
)

// Status of a target in a report.
type Status string

const (
	Changed   Status = "changed"
	Unchanged Status = "unchanged"
	Removed   Status = "removed"
)

// Target is the outcome for one label.
type Target struct {
	Label  string
	Status Status
	// Since is the target's baseline commit ("" if none).
	Since string
	// Commits attributed to the target, newest first.
	Commits []string
	// Files is the sorted union of the files that caused attribution (for a
	// removed target: its gitops files).
	Files []string
}

// Report is the result of Build.
type Report struct {
	Since, Head string
	// Baseline is true iff every universe target has a baseline.
	Baseline bool
	// Targets sorted by label.
	Targets []Target
	// Unmanaged lists merge commits, which are never attributed.
	Unmanaged []string
	// Commits holds every commit referenced by Targets or Unmanaged.
	Commits  map[string]gitx.Commit
	Warnings []string
}

// Input is everything Build needs. Labels are in "//pkg:name" form.
type Input struct {
	Head string
	// Baselines maps label to baseline sha; "" or missing means none.
	Baselines map[string]string
	Universe  []string
	Closures  map[string]graph.Closure
	Mapping   extdeps.Mapping
	// Commits in oldest-baseline..HEAD, newest first (as gitx.Log returns).
	Commits []gitx.Commit
	// InRange maps a baseline sha to the set of commit shas in baseline..HEAD.
	// A baseline without an entry is treated as covering every commit.
	InRange map[string]map[string]bool
	// FileAt returns the content of a workspace-relative path at rev, false
	// if absent. nil disables Level 2.
	FileAt func(rev, path string) ([]byte, bool)
	// Tracked reports whether a path is tracked by git. nil means untracked.
	Tracked     func(path string) bool
	GlobalGlobs []string
	// ModuleBzlFiles are the workspace .bzl files used by MODULE.bazel
	// (use_extension, use_repo_rule) and their transitive loads; they are in
	// no target closure but affect every target.
	ModuleBzlFiles []string
	Gitops         *gitops.Result
}

// Build computes the report.
func Build(in Input) Report {
	r := Report{Head: in.Head, Baseline: true, Commits: map[string]gitx.Commit{}}

	// Classify each non-merge commit's files once.
	var commits []classified
	for _, c := range in.Commits {
		if len(c.Parents) > 1 {
			r.Unmanaged = append(r.Unmanaged, c.SHA)
			r.Commits[c.SHA] = c
			continue
		}
		commits = append(commits, classify(in, c))
	}

	universe := map[string]bool{}
	since, first := "", true
	for _, label := range in.Universe {
		if universe[label] {
			continue
		}
		universe[label] = true
		base := in.Baselines[label]
		if base == "" {
			r.Baseline = false
		}
		if first {
			since, first = base, false
		} else if since != base {
			since = ""
		}
		t := Target{Label: label, Since: base, Status: Changed}
		forceChanged := false
		if base == "" {
			r.Targets = append(r.Targets, t)
			continue
		}
		inRange, known := in.InRange[base]
		if !known {
			// Unknown range: never "unchanged", whatever the commits say.
			forceChanged = true
			r.Warnings = append(r.Warnings, fmt.Sprintf("commit range of baseline %s for %s is unknown; considering all commits", base, label))
		}
		closure, haveClosure := in.Closures[label]
		if !haveClosure {
			r.Warnings = append(r.Warnings, fmt.Sprintf("no closure for %s; attributing every commit in range", label))
		}
		files := map[string]bool{}
		for _, cc := range commits {
			if known && !inRange[cc.c.SHA] {
				continue
			}
			var hit []string
			if haveClosure {
				hit = cc.attributed(closure)
			} else {
				for _, f := range cc.c.Files {
					hit = append(hit, f.Path)
				}
			}
			if len(hit) == 0 && haveClosure {
				continue
			}
			t.Commits = append(t.Commits, cc.c.SHA)
			r.Commits[cc.c.SHA] = cc.c
			for _, f := range hit {
				files[f] = true
			}
		}
		if len(t.Commits) == 0 && !forceChanged {
			t.Status = Unchanged
		}
		t.Files = sortedKeys(files)
		r.Targets = append(r.Targets, t)
	}
	r.Since = since

	if in.Gitops != nil {
		for label, fs := range in.Gitops.Targets {
			if universe[label] {
				continue
			}
			files := append([]string(nil), fs...)
			sort.Strings(files)
			r.Targets = append(r.Targets, Target{Label: label, Status: Removed, Since: in.Baselines[label], Files: files})
		}
	}
	sort.Slice(r.Targets, func(i, j int) bool { return r.Targets[i].Label < r.Targets[j].Label })
	return r
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

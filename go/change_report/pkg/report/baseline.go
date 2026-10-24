package report

import (
	"fmt"
	"sort"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
)

// ResolveBaselines returns the full baseline SHA of each universe target:
// perTarget[label] if present, else since. A baseline that cannot be resolved
// or is not an ancestor of head yields a warning and an empty baseline, which
// Build reports as changed. Keys of perTarget are normalised with label.Parse;
// unparsable keys and keys outside the universe are ignored with a warning.
func ResolveBaselines(repo gitx.Repo, head string, universe []string, since string, perTarget map[string]string) (baselines map[string]string, warnings []string) {
	baselines = map[string]string{}
	inUniverse := map[string]bool{}
	for _, l := range universe {
		inUniverse[l] = true
	}

	over := map[string]string{}
	keys := make([]string, 0, len(perTarget))
	for k := range perTarget {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		l, err := label.Parse(k)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("baseline for %s ignored: %v", k, err))
			continue
		}
		s := l.String()
		if !inUniverse[s] {
			warnings = append(warnings, fmt.Sprintf("baseline for %s ignored: not in --targets universe", s))
			continue
		}
		over[s] = perTarget[k]
	}

	type res struct {
		sha string
		bad string // warning format suffix, "" if ok
	}
	cache := map[string]res{}
	resolve := func(rev, l string) string {
		if rev == "" {
			return ""
		}
		if rev == RootBaseline {
			return RootBaseline
		}
		r, ok := cache[rev]
		if !ok {
			sha, err := repo.RevParse(rev)
			switch {
			case err != nil:
				r.bad = "cannot be resolved"
			default:
				anc, err := repo.IsAncestor(sha, head)
				if err != nil || !anc {
					r.bad = "is not an ancestor of HEAD"
				} else {
					r.sha = sha
				}
			}
			cache[rev] = r
		}
		if r.bad != "" {
			warnings = append(warnings, fmt.Sprintf("baseline %s for %s %s; treating as changed", rev, l, r.bad))
		}
		return r.sha
	}

	for _, l := range universe {
		if _, done := baselines[l]; done {
			continue
		}
		rev := since
		if o, ok := over[l]; ok {
			rev = o
		}
		baselines[l] = resolve(rev, l)
	}
	return baselines, warnings
}

// CollectHistory reads the history needed by Build in one pass: a single Log
// from the oldest baseline (the one that is an ancestor of every other), or
// the full history if no such baseline exists, and one RevSet per distinct
// non-empty baseline. With no baselines it returns no commits.
func CollectHistory(repo gitx.Repo, head string, baselines map[string]string) (commits []gitx.Commit, inRange map[string]map[string]bool, err error) {
	inRange = map[string]map[string]bool{}
	var distinct []string
	seen := map[string]bool{}
	for _, b := range baselines {
		if b != "" && !seen[b] {
			seen[b] = true
			distinct = append(distinct, b)
		}
	}
	if len(distinct) == 0 {
		return nil, inRange, nil
	}
	sort.Strings(distinct)

	// The root baseline covers all history: log from the root and every
	// commit is in its range. Other baselines still get their own RevSet.
	root := seen[RootBaseline]
	oldest := ""
	for _, cand := range distinct {
		if root {
			break
		}
		ok := true
		for _, other := range distinct {
			if other == cand {
				continue
			}
			anc, err := repo.IsAncestor(cand, other)
			if err != nil {
				return nil, nil, err
			}
			if !anc {
				ok = false
				break
			}
		}
		if ok {
			oldest = cand
			break
		}
	}

	commits, err = repo.Log(oldest, head)
	if err != nil {
		return nil, nil, err
	}
	for _, b := range distinct {
		if b == RootBaseline {
			all := make(map[string]bool, len(commits))
			for _, c := range commits {
				all[c.SHA] = true
			}
			inRange[b] = all
			continue
		}
		set, err := repo.RevSet(b, head)
		if err != nil {
			return nil, nil, err
		}
		inRange[b] = set
	}
	return commits, inRange, nil
}

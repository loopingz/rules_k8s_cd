package extdeps

import (
	"encoding/json"
	"regexp"
	"strings"
)

type diffLock struct {
	ModuleExtensions map[string]map[string]struct {
		BzlTransitiveDigest string                     `json:"bzlTransitiveDigest"`
		UsagesDigest        string                     `json:"usagesDigest"`
		GeneratedRepoSpecs  map[string]json.RawMessage `json:"generatedRepoSpecs"`
	} `json:"moduleExtensions"`
	Registry map[string]string `json:"registryFileHashes"`
}

var registryModRe = regexp.MustCompile(`/modules/([^/]+)/([^/]+)/`)

// canon re-encodes JSON with sorted keys and no insignificant whitespace.
func canon(raw json.RawMessage) string {
	var v interface{}
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// ChangedRepos is the Level 2 lockfile diff: the canonical repos whose
// generated spec was added, removed or modified, plus module repos whose
// registry files changed. ok is false if it cannot be determined precisely
// (parse failure, or a change that is not attributable to repos); callers
// then fall back to treating all dependency changes as relevant.
func ChangedRepos(before, after []byte) (repos []string, ok bool) {
	var b, a diffLock
	if json.Unmarshal(before, &b) != nil || json.Unmarshal(after, &a) != nil {
		return nil, false
	}
	// Top-level sections other than the ones handled below must be identical.
	var bm, am map[string]json.RawMessage
	if json.Unmarshal(before, &bm) != nil || json.Unmarshal(after, &am) != nil {
		return nil, false
	}
	for _, k := range []string{"moduleExtensions", "registryFileHashes", "facts", "factsVersions"} {
		delete(bm, k)
		delete(am, k)
	}
	if len(bm) != len(am) {
		return nil, false
	}
	for k, v := range bm {
		if w, found := am[k]; !found || canon(v) != canon(w) {
			return nil, false
		}
	}

	var out []string
	ids := map[string]bool{}
	for id := range b.ModuleExtensions {
		ids[id] = true
	}
	for id := range a.ModuleExtensions {
		ids[id] = true
	}
	for id := range ids {
		prefix, parsed := canonicalPrefix(id)
		if !parsed {
			return nil, false
		}
		bv, av := b.ModuleExtensions[id], a.ModuleExtensions[id]
		variants := map[string]bool{}
		for v := range bv {
			variants[v] = true
		}
		for v := range av {
			variants[v] = true
		}
		for v := range variants {
			x, xok := bv[v]
			y, yok := av[v]
			names := map[string]bool{}
			for n := range x.GeneratedRepoSpecs {
				names[n] = true
			}
			for n := range y.GeneratedRepoSpecs {
				names[n] = true
			}
			digestChanged := xok && yok && (x.BzlTransitiveDigest != y.BzlTransitiveDigest || x.UsagesDigest != y.UsagesDigest)
			for n := range names {
				xs, inX := x.GeneratedRepoSpecs[n]
				ys, inY := y.GeneratedRepoSpecs[n]
				if inX != inY || digestChanged || canon(xs) != canon(ys) {
					out = append(out, prefix+n)
				}
			}
		}
	}

	for k, bh := range b.Registry {
		if ah, found := a.Registry[k]; !found || ah != bh {
			if !addRegistryRepos(&out, k) {
				return nil, false
			}
		}
	}
	for k := range a.Registry {
		if _, found := b.Registry[k]; !found {
			if !addRegistryRepos(&out, k) {
				return nil, false
			}
		}
	}
	return uniqSorted(out), true
}

func addRegistryRepos(out *[]string, key string) bool {
	m := registryModRe.FindStringSubmatch(key)
	if m == nil || strings.Contains(m[1], "+") {
		return false
	}
	*out = append(*out, m[1]+"+", m[1]+"+"+m[2])
	return true
}

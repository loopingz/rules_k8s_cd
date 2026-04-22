package main

import (
	"fmt"
	"strings"
)

// Selector matches a subset of resource identity fields. A missing key means
// "don't match on this field". Allowed keys: apiVersion, kind, name, namespace.
// All present keys must match (AND).
type Selector map[string]string

var allowedSelectorKeys = map[string]struct{}{
	"apiVersion": {},
	"kind":       {},
	"name":       {},
	"namespace":  {},
}

// ParseSelector parses a comma-separated list of key=value pairs.
// Unknown keys are rejected so typos don't silently become no-ops.
func ParseSelector(s string) (Selector, error) {
	out := Selector{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			return nil, fmt.Errorf("selector %q: expected key=value", part)
		}
		k := strings.TrimSpace(part[:eq])
		v := strings.TrimSpace(part[eq+1:])
		if _, ok := allowedSelectorKeys[k]; !ok {
			return nil, fmt.Errorf("selector %q: unknown key %q (allowed: apiVersion, kind, name, namespace)", part, k)
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("selector %q: empty", s)
	}
	return out, nil
}

// ApplyExcludes returns a new slice containing only the documents that do
// not match any of the given selectors.
func ApplyExcludes(docs []map[string]any, excludes []Selector) []map[string]any {
	if len(excludes) == 0 {
		return docs
	}
	out := make([]map[string]any, 0, len(docs))
	for _, d := range docs {
		drop := false
		for _, sel := range excludes {
			if matchesSelector(d, sel) {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, d)
		}
	}
	return out
}

func matchesSelector(doc map[string]any, sel Selector) bool {
	for k, v := range sel {
		var got string
		switch k {
		case "apiVersion":
			got, _ = doc["apiVersion"].(string)
		case "kind":
			got, _ = doc["kind"].(string)
		case "name":
			if md, ok := doc["metadata"].(map[string]any); ok {
				got, _ = md["name"].(string)
			}
		case "namespace":
			if md, ok := doc["metadata"].(map[string]any); ok {
				got, _ = md["namespace"].(string)
			}
		}
		if got != v {
			return false
		}
	}
	return true
}

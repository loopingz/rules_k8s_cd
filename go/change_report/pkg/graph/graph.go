// Package graph computes the closure of a Bazel target from `bazel query` nodes.
package graph

import (
	"sort"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
)

// Graph holds query nodes keyed by canonical label ("@@//pkg:name" for the
// main repo) and the load graph of workspace .bzl/BUILD files.
type Graph struct {
	Nodes map[string]bazelq.Node
	Loads map[string][]string
}

// Closure is everything a target depends on. All slices are sorted and non-nil.
type Closure struct {
	Files, Packages, Repos []string
}

// key normalizes a label to its canonical "@@repo//pkg:name" form. Unparseable
// labels are returned unchanged.
func key(s string) string {
	l, err := label.Parse(s)
	if err != nil {
		return s
	}
	return "@@" + l.Repo + "//" + l.Pkg + ":" + l.Name
}

// New builds a Graph from query nodes.
func New(nodes []bazelq.Node) *Graph {
	g := &Graph{Nodes: make(map[string]bazelq.Node, len(nodes)), Loads: map[string][]string{}}
	g.add(nodes)
	return g
}

func (g *Graph) add(nodes []bazelq.Node) {
	for _, n := range nodes {
		g.Nodes[key(n.Label)] = n
	}
}

// ReplacePackages drops all main-repo nodes in pkgs, then adds nodes.
func (g *Graph) ReplacePackages(pkgs []string, nodes []bazelq.Node) {
	drop := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		drop[p] = true
	}
	for k, n := range g.Nodes {
		l, err := label.Parse(n.Label)
		if err == nil && l.IsMain() && drop[l.Pkg] {
			delete(g.Nodes, k)
		}
	}
	g.add(nodes)
}

// SetLoads records the workspace files loaded by path.
func (g *Graph) SetLoads(path string, loads []string) {
	g.Loads[path] = loads
}

// Closure computes the closure of target. Files holds main-repo paths only:
// source files, plus each package's BUILD file and its transitive loads.
// External labels only contribute their repo. Labels absent from Nodes are
// handled conservatively (main-repo ones still contribute their file path).
func (g *Graph) Closure(target string, buildFile func(pkg string) string) Closure {
	files := map[string]bool{}
	pkgs := map[string]bool{}
	repos := map[string]bool{}
	seen := map[string]bool{}

	stack := []string{key(target)}
	for len(stack) > 0 {
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[k] {
			continue
		}
		seen[k] = true
		l, err := label.Parse(k)
		if err != nil {
			continue
		}
		if !l.IsMain() {
			repos[l.Repo] = true
		} else {
			pkgs[l.Pkg] = true
		}
		n, ok := g.Nodes[k]
		if l.IsMain() && (!ok || n.Kind == "source") {
			files[l.Path()] = true
		}
		if !ok {
			continue
		}
		switch n.Kind {
		case "rule":
			for _, in := range n.Inputs {
				stack = append(stack, key(in))
			}
		case "generated":
			if n.GeneratingRule != "" {
				stack = append(stack, key(n.GeneratingRule))
			}
		}
	}

	for p := range pkgs {
		bf := buildFile(p)
		g.addLoadClosure(bf, files)
	}
	return Closure{Files: sortedKeys(files), Packages: sortedKeys(pkgs), Repos: sortedKeys(repos)}
}

// LoadClosure returns the sorted workspace files reachable from starts
// (included) through Loads.
func (g *Graph) LoadClosure(starts []string) []string {
	files := map[string]bool{}
	for _, s := range starts {
		g.addLoadClosure(s, files)
	}
	return sortedKeys(files)
}

func (g *Graph) addLoadClosure(start string, files map[string]bool) {
	stack := []string{start}
	seen := map[string]bool{}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[p] {
			continue
		}
		seen[p] = true
		files[p] = true
		stack = append(stack, g.Loads[p]...)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

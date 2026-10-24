package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
)

func isBuildFile(p string) bool {
	b := path.Base(p)
	return b == "BUILD" || b == "BUILD.bazel"
}

// pkgDir returns the directory of p as a package name ("" for the root).
func pkgDir(p string) string {
	d := path.Dir(p)
	if d == "." {
		return ""
	}
	return d
}

// isGlobalFile reports whether p influences the whole graph.
func isGlobalFile(p string) bool {
	switch {
	case p == "MODULE.bazel", p == "MODULE.bazel.lock", p == ".bazelrc", p == ".bazelversion":
		return true
	case strings.HasPrefix(p, "WORKSPACE") && !strings.Contains(p, "/"):
		return true
	}
	return strings.HasSuffix(p, ".bzl")
}

// GlobalKey hashes everything that can change the graph of every package:
// FormatVersion, the universe expression, the Bazel version and the blob SHAs
// of MODULE.bazel, MODULE.bazel.lock, WORKSPACE*, .bazelrc, .bazelversion and
// every *.bzl file, in sorted path order.
func GlobalKey(universe, bazelVersion string, tree map[string]string) string {
	var paths []string
	for p := range tree {
		if isGlobalFile(p) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	h := sha256.New()
	fmt.Fprintf(h, "format %d\nuniverse %q\nversion %q\n", FormatVersion, universe, bazelVersion)
	for _, p := range paths {
		fmt.Fprintf(h, "%s %q\n", tree[p], p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Packages returns the directories (packages) containing a BUILD or
// BUILD.bazel file; the root package is "".
func Packages(tree map[string]string) map[string]bool {
	pkgs := map[string]bool{}
	for p := range tree {
		if isBuildFile(p) {
			pkgs[pkgDir(p)] = true
		}
	}
	return pkgs
}

// owner returns the nearest enclosing package of file p, or false if none.
func owner(p string, pkgs map[string]bool) (string, bool) {
	d := p
	for {
		d = pkgDir(d)
		if pkgs[d] {
			return d, true
		}
		if d == "" {
			return "", false
		}
	}
}

// PackageKey hashes the BUILD file blob(s) of pkg plus the sorted paths of the
// files whose nearest enclosing package is pkg. File contents other than the
// BUILD file are deliberately left out: editing a file cannot change the
// graph, but adding or removing one can (globs).
func PackageKey(pkg string, tree map[string]string, pkgs map[string]bool) string {
	var owned []string
	for p := range tree {
		if o, ok := owner(p, pkgs); ok && o == pkg {
			owned = append(owned, p)
		}
	}
	return packageKey(pkg, owned, tree)
}

// packageKeys computes PackageKey for every package in one pass over tree.
func packageKeys(tree map[string]string, pkgs map[string]bool) map[string]string {
	owned := make(map[string][]string, len(pkgs))
	for p := range tree {
		if o, ok := owner(p, pkgs); ok {
			owned[o] = append(owned[o], p)
		}
	}
	keys := make(map[string]string, len(pkgs))
	for pkg := range pkgs {
		keys[pkg] = packageKey(pkg, owned[pkg], tree)
	}
	return keys
}

func packageKey(pkg string, owned []string, tree map[string]string) string {
	sort.Strings(owned)
	h := sha256.New()
	fmt.Fprintf(h, "package %q\n", pkg)
	for _, name := range []string{"BUILD.bazel", "BUILD"} {
		bf := path.Join(pkg, name)
		if sha, ok := tree[bf]; ok {
			fmt.Fprintf(h, "build %s %q\n", sha, bf)
		}
	}
	for _, p := range owned {
		fmt.Fprintf(h, "file %q\n", p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

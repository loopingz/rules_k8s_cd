package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
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
	b := path.Base(p)
	switch {
	case p == "MODULE.bazel", p == "MODULE.bazel.lock", p == ".bazelversion", p == ".bazelignore", b == "REPO.bazel":
		return true
	case strings.HasPrefix(p, "WORKSPACE") && !strings.Contains(p, "/"):
		return true
	}
	return strings.HasSuffix(p, ".bzl") || strings.HasSuffix(b, ".bazelrc") || strings.HasSuffix(b, ".MODULE.bazel")
}

// isInputFile reports whether p can change the external dependency mapping
// (module files and the dependency-looking files extensions typically read).
// Such files are hashed by working-tree content, since they may be untracked
// (MODULE.bazel.lock) or edited without changing any package key.
func isInputFile(p string) bool {
	b := path.Base(p)
	switch b {
	case "MODULE.bazel", "go.mod", "go.sum", "package.json", "pnpm-lock.yaml", "package-lock.json", "yarn.lock", "Chart.yaml", "Chart.lock":
		return true
	}
	if ok, _ := path.Match("requirements*.txt", b); ok {
		return true
	}
	return strings.HasSuffix(b, ".lock") || strings.HasSuffix(b, ".MODULE.bazel")
}

// inputPaths returns the sorted union of the input files of tree, the root
// MODULE.bazel(.lock) (tracked or not) and extra (the files of the stored
// external dependency mapping).
func inputPaths(tree map[string]string, extra []string) []string {
	set := map[string]bool{"MODULE.bazel": true, "MODULE.bazel.lock": true}
	for p := range tree {
		if isInputFile(p) {
			set[p] = true
		}
	}
	for _, p := range extra {
		set[p] = true
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// inputsDigest hashes the working-tree content of paths (absent files hash
// as absent).
func inputsDigest(ws string, paths []string) string {
	h := sha256.New()
	for _, p := range paths {
		data, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(p)))
		if err != nil {
			fmt.Fprintf(h, "absent %q\n", p)
			continue
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(h, "file %x %q\n", sum, p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// GlobalKey hashes everything that can change the graph of every package:
// FormatVersion, the universe expression, the Bazel version and the blob SHAs
// of MODULE.bazel, MODULE.bazel.lock, *.MODULE.bazel, WORKSPACE*, *.bazelrc,
// .bazelversion, .bazelignore, REPO.bazel and every *.bzl file, in sorted path
// order. Working-tree inputs of the dependency mapping are covered separately
// (inputsDigest).
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

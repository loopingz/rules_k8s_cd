package graph

import (
	"path"
	"regexp"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
)

var loadRE = regexp.MustCompile(`\bload\s*\(\s*["']([^"']+)["']`)

// ParseLoads returns the workspace paths of the files loaded by src, a BUILD
// or .bzl file of package pkg ("" for the root package). Relative loads
// (":x.bzl") resolve against pkg, as Bazel does, even for a .bzl file in a
// subdirectory of its package. Loads from external repositories are dropped,
// except repos named in mainRepoNames (the workspace's own module name or
// repo_name), which refer back to the workspace. The regexp may over-match
// (e.g. in comments), which only adds files and is safe.
func ParseLoads(src []byte, pkg string, mainRepoNames []string) []string {
	var out []string
	for _, m := range loadRE.FindAllSubmatch(src, -1) {
		s := string(m[1])
		if strings.HasPrefix(s, ":") {
			out = append(out, path.Join(pkg, s[1:]))
			continue
		}
		l, err := label.Parse(s)
		if err != nil {
			continue
		}
		if !l.IsMain() && !contains(mainRepoNames, l.Repo) {
			continue
		}
		out = append(out, l.Path())
	}
	return out
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

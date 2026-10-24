package graph

import (
	"path"
	"regexp"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
)

var loadRE = regexp.MustCompile(`load\(\s*["']([^"']+)["']`)

// ParseLoads returns the workspace paths of the files loaded by src (located
// at workspace path fromPath). Loads from external repositories are dropped,
// except repos named in mainRepoNames (the workspace's own module name),
// which refer back to the workspace. The regexp may over-match (e.g. in
// comments), which only adds files and is safe.
func ParseLoads(src []byte, fromPath string, mainRepoNames []string) []string {
	var out []string
	dir := path.Dir(fromPath)
	if dir == "." {
		dir = ""
	}
	for _, m := range loadRE.FindAllSubmatch(src, -1) {
		s := string(m[1])
		if strings.HasPrefix(s, ":") {
			out = append(out, path.Join(dir, s[1:]))
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

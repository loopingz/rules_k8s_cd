package extdeps

import (
	"regexp"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/stmt"
)

// toolchainTagRe matches extension tags that typically set up toolchains,
// whose repos targets reach through toolchain resolution, which `bazel query`
// does not show: their edits stay global.
var toolchainTagRe = regexp.MustCompile(`(?i)toolchain|download|sdk|host|register|platform`)

var identRe = regexp.MustCompile(`^\s*([A-Za-z_]\w*)\s*$`)

// ModuleEditScope tells which external repositories an edit of MODULE.bazel
// can change, statement by statement, when it can tell: ok is false when some
// changed statement may affect anything (bazel_dep, overrides, toolchains,
// register_toolchains, include, constants, unknown calls).
//
// A changed tag of an extension (`oci.pull(...)`) or its `use_extension` maps
// to the extension's repos ("<module>++<ext>+*", "+<ext>+*"
// for the main module); a changed call of a use_repo_rule (`http_file(name =
// "x", ...)`), or its `use_repo_rule` definition, to its repos ("*+x"), unless
// register_toolchains or register_execution_platforms names them. A changed
// `use_repo` changes no repo. mainNames are the apparent names of
// the main repository (MainRepoNames).
func ModuleEditScope(before, after []byte, mainNames []string) (patterns []string, ok bool) {
	sb, sa := stmt.Split(string(before)), stmt.Split(string(after))
	repoNames := map[string]string{} // bazel_dep repo_name -> module name
	exts := map[string]string{}      // variable -> repo pattern ("" unknown)
	rules := map[string]bool{}       // use_repo_rule variables
	for _, s := range append(append([]stmt.Stmt(nil), sb...), sa...) {
		if s.Call == "bazel_dep" {
			if r := s.Kwarg("repo_name"); r != "" {
				repoNames[r] = s.Kwarg("name")
			}
		}
	}
	for _, s := range append(append([]stmt.Stmt(nil), sb...), sa...) {
		switch {
		case s.Call == "use_extension" && s.Assign != "":
			p := extPattern(s, repoNames, mainNames)
			if prev, seen := exts[s.Assign]; seen && prev != p {
				p = "" // redefined differently: unknown
			}
			exts[s.Assign] = p
		case s.Call == "use_repo_rule" && s.Assign != "":
			rules[s.Assign] = true
		}
	}
	registered := func(repo string) bool {
		for _, s := range append(append([]stmt.Stmt(nil), sb...), sa...) {
			if (s.Call == "register_toolchains" || s.Call == "register_execution_platforms") && strings.Contains(s.Args, "@"+repo+"//") {
				return true
			}
		}
		return false
	}
	seen := map[string]bool{}
	add := func(p string) bool {
		if p == "" {
			return false
		}
		if !seen[p] {
			seen[p] = true
			patterns = append(patterns, p)
		}
		return true
	}
	for _, s := range stmt.Changed(sb, sa) {
		switch {
		case s.Call == "use_extension" && s.Assign != "":
			if !add(extPattern(s, repoNames, mainNames)) {
				return nil, false
			}
		case strings.Contains(s.Call, ".") && s.Assign == "":
			v, tag, _ := strings.Cut(s.Call, ".")
			if toolchainTagRe.MatchString(tag) || !add(exts[v]) {
				return nil, false
			}
		case s.Call == "use_repo" && s.Assign == "":
			// Only which repos the main module can name: a target that starts
			// (or stops) using one changes its BUILD file, attributed already.
			args := splitTop(s.Args)
			if len(args) == 0 || identRe.FindStringSubmatch(args[0]) == nil {
				return nil, false
			}
		case s.Call == "use_repo_rule" && s.Assign != "":
			// The repos of its calls
			for _, c := range append(append([]stmt.Stmt(nil), sb...), sa...) {
				if c.Call != s.Assign || c.Assign != "" {
					continue
				}
				name := c.Kwarg("name")
				if name == "" || registered(name) || !add("*+"+name) {
					return nil, false
				}
			}
		case rules[s.Call] && s.Assign == "":
			name := s.Kwarg("name")
			if name == "" || registered(name) || !add("*+"+name) {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return patterns, true
}

// extPattern is the canonical repo pattern of a use_extension statement's
// repos, "" when unknown (isolated usage, unparsable label).
func extPattern(s stmt.Stmt, repoNames map[string]string, mainNames []string) string {
	file, name := s.Positional(0), s.Positional(1)
	if file == "" || name == "" || strings.Contains(s.Args, "isolate") {
		return ""
	}
	module := ""
	switch {
	case strings.HasPrefix(file, "//"), strings.HasPrefix(file, "@//"), strings.HasPrefix(file, "@@//"), strings.HasPrefix(file, ":"):
	case strings.HasPrefix(file, "@@"):
		return "" // canonical labels are not expected in MODULE.bazel
	case strings.HasPrefix(file, "@"):
		repo, _, found := strings.Cut(file[1:], "//")
		if !found || repo == "" {
			return ""
		}
		module = repo
		for _, m := range mainNames {
			if m == repo {
				module = ""
			}
		}
		if n, ok := repoNames[repo]; ok && module != "" {
			module = n
		}
	default:
		return ""
	}
	if module == "" {
		return "+" + name + "+*"
	}
	return module + "++" + name + "+*"
}

func splitTop(a string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(a); i++ {
		switch a[i] {
		case '"', '\'':
			i = skipString(a, i) - 1
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, a[start:i])
				start = i + 1
			}
		}
	}
	if strings.TrimSpace(a[start:]) != "" {
		out = append(out, a[start:])
	}
	return out
}

var overrideCalls = map[string]string{
	"bazel_dep":                 "name",
	"git_override":              "module_name",
	"archive_override":          "module_name",
	"single_version_override":   "module_name",
	"multiple_version_override": "module_name",
}

// ModuleDepChanges returns the modules whose bazel_dep or override
// (git_override, archive_override, single/multiple_version_override) changed
// between two MODULE.bazel contents. ok is false when anything else changed,
// or a changed call names no module. A module's new version or commit changes
// its own repos ("<name>+", "<name>++*"); the versions it makes other modules
// resolve to show in the lockfile's registry section.
func ModuleDepChanges(before, after []byte) (modules []string, ok bool) {
	seen := map[string]bool{}
	for _, s := range stmt.Changed(stmt.Split(string(before)), stmt.Split(string(after))) {
		key, known := overrideCalls[s.Call]
		if !known || s.Assign != "" {
			return nil, false
		}
		name := s.Kwarg(key)
		if name == "" {
			return nil, false
		}
		if !seen[name] {
			seen[name] = true
			modules = append(modules, name)
		}
	}
	return modules, true
}

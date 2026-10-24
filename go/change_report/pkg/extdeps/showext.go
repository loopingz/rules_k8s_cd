package extdeps

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
)

var quotedRe = regexp.MustCompile(`"([^"\\]*)"`)

// companionGroups lists files that are read together; mapping one maps all
// siblings in the same directory.
var companionGroups = [][]string{
	{"go.mod", "go.sum"},
	{"go.work", "go.work.sum"},
	{"package.json", "pnpm-lock.yaml", "package-lock.json", "yarn.lock"},
}

func addWithCompanions(files map[string]bool, p string) {
	files[p] = true
	dir, base := path.Split(p)
	for _, g := range companionGroups {
		for _, n := range g {
			if n == base {
				for _, s := range g {
					files[dir+s] = true
				}
			}
		}
	}
}

// FromShowExtension parses `bazel mod show_extension` output. Files come from
// main-repo labels in the root module's tag lines; repos from each extension's
// "Fetched repositories:" list, mapped to canonical names.
func FromShowExtension(out []byte) Mapping {
	m := Mapping{FileRepos: map[string][]string{}}
	var (
		repos   []string // current extension's canonical repos
		prefix  string
		inRepos bool
		inRoot  bool
		inUse   bool
		files   = map[string]bool{}
	)
	flush := func() {
		for f := range files {
			m.FileRepos[f] = append(m.FileRepos[f], repos...)
		}
		files = map[string]bool{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "## Usage in "):
			inRepos, inUse = false, false
			inRoot = strings.HasPrefix(line, "## Usage in <root>")
			continue
		case strings.HasPrefix(line, "## "):
			flush()
			id := strings.TrimSuffix(strings.TrimSpace(line[3:]), ":")
			prefix, _ = canonicalPrefix(id)
			repos = nil
			inRepos, inRoot, inUse = false, false, false
			continue
		case strings.HasPrefix(line, "Fetched repositories:"):
			inRepos = true
			continue
		}
		if inRepos {
			t := strings.TrimSpace(line)
			if rest, ok := strings.CutPrefix(t, "- "); ok {
				if f := strings.Fields(rest); len(f) > 0 {
					repos = append(repos, prefix+f[0])
				}
			} else if t != "" {
				inRepos = false
			}
			continue
		}
		if !inRoot {
			continue
		}
		t := strings.TrimSpace(line)
		if inUse {
			if strings.HasPrefix(t, ")") {
				inUse = false
			}
			continue
		}
		if strings.HasPrefix(t, "use_repo(") {
			inUse = !strings.Contains(t, ")")
			continue
		}
		for _, q := range quotedRe.FindAllStringSubmatch(t, -1) {
			if p, ok := labelPath(q[1]); ok {
				addWithCompanions(files, p)
			}
		}
	}
	flush()
	for f := range m.FileRepos {
		m.FileRepos[f] = uniqSorted(m.FileRepos[f])
	}
	return m
}

var useExtRe = regexp.MustCompile(`use_extension\(\s*(?:\w+\s*=\s*)?"([^"]+)"\s*,\s*(?:\w+\s*=\s*)?"([^"]+)"`)

func stripComments(mb []byte) string {
	var b strings.Builder
	for _, l := range strings.Split(string(mb), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String()
}

var (
	moduleCallRe = regexp.MustCompile(`(?s)\bmodule\s*\(([^)]*)\)`)
	moduleAttrRe = regexp.MustCompile(`\b(name|repo_name)\s*=\s*"([^"]+)"`)
)

// MainRepoNames returns the apparent names the root module gives itself: the
// module() name and repo_name, if set. Labels "@<name>//..." refer to the
// main repository.
func MainRepoNames(moduleBazel []byte) []string {
	m := moduleCallRe.FindStringSubmatch(stripComments(moduleBazel))
	if m == nil {
		return nil
	}
	var out []string
	for _, a := range moduleAttrRe.FindAllStringSubmatch(m[1], -1) {
		out = append(out, a[2])
	}
	return uniqSorted(out)
}

// rootID rewrites a bzl label that refers to the root module ("//x",
// "@//x" or "@<self>//x" for a name in selves) to "@@//x", the only form
// `bazel mod show_extension` accepts for it. Other labels are returned
// unchanged.
func rootID(bzl string, selves []string) string {
	switch {
	case strings.HasPrefix(bzl, "@@//"):
		return bzl
	case strings.HasPrefix(bzl, "//"):
		return "@@" + bzl
	case strings.HasPrefix(bzl, "@//"):
		return "@" + bzl
	}
	for _, self := range selves {
		if strings.HasPrefix(bzl, "@"+self+"//") {
			return "@@" + bzl[len("@"+self):]
		}
	}
	return bzl
}

// RootExtensions returns "<bzl>%<name>" ids for the use_extension calls in
// MODULE.bazel, in the apparent-label form `bazel mod show_extension` accepts.
func RootExtensions(moduleBazel []byte) []string {
	var ids []string
	seen := map[string]bool{}
	src := stripComments(moduleBazel)
	selves := MainRepoNames(moduleBazel)
	for _, m := range useExtRe.FindAllStringSubmatch(src, -1) {
		id := rootID(m[1], selves) + "%" + m[2]
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

var (
	localRe  = regexp.MustCompile(`(?s)\b(?:local_path_override|(?:new_)?local_repository)\s*\(([^)]*)\)`)
	pathAttr = regexp.MustCompile(`\bpath\s*=\s*"([^"]*)"`)
	depRe    = regexp.MustCompile(`\bbazel_dep\(\s*name\s*=\s*"([^"]+)"`)
)

var moduleBzlCallRe = regexp.MustCompile(`(?s)\b(?:use_extension|use_repo_rule)\s*\(([^)]*)\)`)

// ModuleBzlPaths returns the sorted workspace paths of the main-repo .bzl
// files named by use_extension and use_repo_rule calls in a MODULE.bazel (or
// included *.MODULE.bazel) file. Main-repo labels are "//x", "@//x", "@@//x",
// "@<name>//x" for a name in mainNames, and ":x.bzl" (root package). Every
// quoted ".bzl" string of a call is considered, whatever its argument
// position: over-matching only makes more files global.
func ModuleBzlPaths(src []byte, mainNames []string) []string {
	var out []string
	for _, call := range moduleBzlCallRe.FindAllStringSubmatch(stripComments(src), -1) {
		for _, q := range quotedRe.FindAllStringSubmatch(call[1], -1) {
			s := q[1]
			if !strings.HasSuffix(s, ".bzl") {
				continue
			}
			if rest, ok := strings.CutPrefix(s, ":"); ok {
				out = append(out, rest)
				continue
			}
			l, err := label.Parse(s)
			if err != nil {
				continue
			}
			if l.IsMain() || slices.Contains(mainNames, l.Repo) {
				out = append(out, l.Path())
			}
		}
	}
	return uniqSorted(out)
}

// LocalPaths returns the `path` of local_path_override, local_repository and
// new_local_repository calls.
func LocalPaths(moduleBazel []byte) []string {
	var out []string
	for _, m := range localRe.FindAllStringSubmatch(stripComments(moduleBazel), -1) {
		if p := pathAttr.FindStringSubmatch(m[1]); p != nil {
			out = append(out, p[1])
		}
	}
	return uniqSorted(out)
}

// ModuleRepos returns the canonical repos ("<name>+") of bazel_dep modules.
func ModuleRepos(moduleBazel []byte) []string {
	var out []string
	for _, m := range depRe.FindAllStringSubmatch(stripComments(moduleBazel), -1) {
		out = append(out, m[1]+"+")
	}
	return uniqSorted(out)
}

// Load combines the lockfile mapping with `bazel mod show_extension` for the
// root module's extensions. If lockfile is empty it first runs
// `bazel mod deps --lockfile_mode=update` and reads <ws>/MODULE.bazel.lock.
// Extension ids for which show_extension fails are skipped and returned as
// warnings (the caller's unmapped-dependency-file safety net covers them).
func Load(ws string, run bazelq.Runner, lockfile []byte, moduleBazel []byte) (Mapping, []string, error) {
	if len(lockfile) == 0 {
		if _, err := run.Run("mod", "deps", "--lockfile_mode=update"); err != nil {
			return Mapping{}, nil, fmt.Errorf("generating lockfile: %w", err)
		}
		var err error
		if lockfile, err = os.ReadFile(filepath.Join(ws, "MODULE.bazel.lock")); err != nil {
			return Mapping{}, nil, fmt.Errorf("reading lockfile: %w", err)
		}
	}
	m, err := FromLockfile(lockfile)
	if err != nil {
		return Mapping{}, nil, fmt.Errorf("parsing lockfile: %w", err)
	}
	var warns []string
	if ids := RootExtensions(moduleBazel); len(ids) > 0 {
		out, err := run.Run(append([]string{"mod", "show_extension"}, ids...)...)
		if err == nil {
			m = m.Merge(FromShowExtension(out))
		} else if len(ids) == 1 {
			warns = append(warns, ids[0])
		} else {
			for _, id := range ids {
				if out, err := run.Run("mod", "show_extension", id); err != nil {
					warns = append(warns, id)
				} else {
					m = m.Merge(FromShowExtension(out))
				}
			}
		}
	}
	m.ModuleRepos = ModuleRepos(moduleBazel)
	m.LocalPaths = LocalPaths(moduleBazel)
	if m.FileRepos == nil {
		m.FileRepos = map[string][]string{}
	}
	return m, warns, nil
}

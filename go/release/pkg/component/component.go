// Package component discovers release_component declarations with
// `bazel query` and resolves each component's current version and the commit
// its changes are attributed from.
package component

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/label"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
	"loopingz.com/rules_k8s_cd/release/pkg/tagfmt"
)

const (
	defaultTagFormat = "{name}@v{version}"
	defaultInitial   = "0.1.0"
)

// Component is a release_component declaration. Paths are workspace-relative
// and labels are in display form (//pkg:name).
type Component struct {
	Label, Name, Package string
	Targets              []string
	TargetsQuery         string
	VersionFile          string
	Changelog            string
	Untagged, TagFormat  string
	Initial              semver.Version
	Ignore               []string          // workspace-relative paths
	BumpFiles            map[string]string // workspace-relative path -> regex
	ReleaseTargets       []string
}

// Discover queries the release_component rules in scope (default //...) and
// returns them sorted by Name. Names must be unique.
func Discover(r bazelq.Runner, scope string) ([]Component, error) {
	if scope == "" {
		scope = "//..."
	}
	nodes, err := bazelq.QueryWithAttrs(r, fmt.Sprintf("kind(\"_release_component rule\", %s)", scope))
	if err != nil {
		return nil, err
	}
	var out []Component
	byName := map[string]string{}
	for _, n := range nodes {
		if n.Kind != "rule" || n.RuleClass != "_release_component" {
			continue
		}
		c, err := fromNode(n)
		if err != nil {
			return nil, err
		}
		if prev, dup := byName[c.Name]; dup {
			return nil, fmt.Errorf("duplicate release_component name %q: %s and %s", c.Name, prev, c.Label)
		}
		byName[c.Name] = c.Label
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func fromNode(n bazelq.Node) (Component, error) {
	l, err := label.Parse(n.Label)
	if err != nil {
		return Component{}, err
	}
	c := Component{Label: l.String(), Name: l.Name, Package: l.Pkg}
	fail := func(err error) (Component, error) {
		return Component{}, fmt.Errorf("release_component %s: %w", c.Label, err)
	}
	str := func(name string) string { return n.Attrs[name].String }
	// Task 4 stores version_file and changelog workspace-relative; bump_files
	// keys and ignore globs are package-relative.
	inPkg := func(p string) string {
		if c.Package == "" {
			return p
		}
		return path.Join(c.Package, p)
	}

	c.VersionFile = str("version_file")
	c.Changelog = str("changelog")
	c.TargetsQuery = str("targets_query")
	c.Untagged = str("untagged_version")
	c.TagFormat = str("tag_format")
	if c.TagFormat == "" {
		c.TagFormat = defaultTagFormat
	}
	if err := tagfmt.Validate(c.TagFormat, c.Name); err != nil {
		return fail(err)
	}
	initial := str("initial_version")
	if initial == "" {
		initial = defaultInitial
	}
	if c.Initial, err = semver.Parse(initial); err != nil {
		return fail(fmt.Errorf("initial_version: %w", err))
	}
	if c.Targets, err = labels(n.Attrs["targets"].StringList); err != nil {
		return fail(err)
	}
	if c.ReleaseTargets, err = labels(n.Attrs["release_targets"].StringList); err != nil {
		return fail(err)
	}
	for _, g := range n.Attrs["ignore"].StringList {
		c.Ignore = append(c.Ignore, inPkg(g))
	}
	for k, v := range n.Attrs["bump_files"].StringDict {
		if c.BumpFiles == nil {
			c.BumpFiles = map[string]string{}
		}
		c.BumpFiles[inPkg(k)] = v
	}
	return c, nil
}

// labels normalises labels to display form (//pkg:name).
func labels(in []string) ([]string, error) {
	var out []string
	for _, s := range in {
		l, err := label.Parse(s)
		if err != nil {
			return nil, err
		}
		out = append(out, l.String())
	}
	return out, nil
}

// Tag returns the git tag of version v.
func (c Component) Tag(v semver.Version) string {
	return tagfmt.Format(c.TagFormat, c.Name, v)
}

// ParseTag returns the version of tag if it is a tag of this component.
func (c Component) ParseTag(tag string) (semver.Version, bool) {
	return tagfmt.Parser(c.TagFormat, c.Name)(tag)
}

// UniverseExpr is the Bazel query expression selecting the component's targets.
func (c Component) UniverseExpr() string {
	if c.TargetsQuery != "" {
		return "(" + c.TargetsQuery + ")"
	}
	return "set(" + strings.Join(c.Targets, " ") + ")"
}

// State is a component's current release state.
type State struct {
	Current *semver.Version // nil = never released
	// Source is "version_file", "tag" or "none".
	Source string
	// BaselineSHA is the commit changes are attributed from: the tag commit
	// of Current, else of the highest tag, else report.RootBaseline.
	BaselineSHA string
}

// ListTags returns all tags of the repository.
func ListTags(repo gitx.Repo) ([]string, error) {
	return repo.Tags()
}

// Current resolves the component's version: the VERSION file at HEAD if
// present, else the highest tag of the component, else none. tags is the
// repository's tag list (see ListTags).
func Current(repo gitx.Repo, c Component, tags []string) (State, []string, error) {
	parse := tagfmt.Parser(c.TagFormat, c.Name)
	var (
		highest    semver.Version
		highestTag string
		owned      = map[string]bool{}
	)
	for _, t := range tags {
		v, ok := parse(t)
		if !ok {
			continue
		}
		owned[t] = true
		if highestTag == "" || highest.Less(v) {
			highest, highestTag = v, t
		}
	}
	// commitOf resolves an owned tag; false if it is absent.
	commitOf := func(tag string) (string, bool, error) {
		if !owned[tag] {
			return "", false, nil
		}
		return repo.TagCommit(tag)
	}
	// latest is the baseline of the highest tag, or the root.
	latest := func() (string, error) {
		if highestTag == "" {
			return report.RootBaseline, nil
		}
		sha, ok, err := commitOf(highestTag)
		if err != nil {
			return "", err
		}
		if !ok {
			return report.RootBaseline, nil
		}
		return sha, nil
	}

	if c.VersionFile != "" {
		b, ok, err := repo.Show("HEAD", c.VersionFile)
		if err != nil {
			return State{}, nil, err
		}
		if ok {
			v, err := semver.Parse(strings.TrimSpace(string(b)))
			if err != nil {
				return State{}, nil, fmt.Errorf("%s: %w", c.VersionFile, err)
			}
			st := State{Current: &v, Source: "version_file"}
			tag := c.Tag(v)
			sha, found, err := commitOf(tag)
			if err != nil {
				return State{}, nil, err
			}
			if found {
				st.BaselineSHA = sha
				return st, nil, nil
			}
			if st.BaselineSHA, err = latest(); err != nil {
				return State{}, nil, err
			}
			return st, []string{fmt.Sprintf("VERSION %s of %s has no tag %s; using latest tag as baseline", v, c.Name, tag)}, nil
		}
	}

	if highestTag == "" {
		return State{Source: "none", BaselineSHA: report.RootBaseline}, nil, nil
	}
	sha, err := latest()
	if err != nil {
		return State{}, nil, err
	}
	return State{Current: &highest, Source: "tag", BaselineSHA: sha}, nil, nil
}

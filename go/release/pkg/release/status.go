// Package release orchestrates releases: `pr` maintains one release pull
// request per component, `tag` cuts the release (tag, forge release, release
// targets) once a release PR is merged, and `status` reports the plan
// without writing anything.
//
// Every command is safe to re-run, and with Env.DryRun nothing is pushed,
// tagged, written to the forge or run.
package release

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/release/pkg/attrib"
	"loopingz.com/rules_k8s_cd/release/pkg/changelog"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
	"loopingz.com/rules_k8s_cd/release/pkg/conventional"
	"loopingz.com/rules_k8s_cd/release/pkg/forge"
	"loopingz.com/rules_k8s_cd/release/pkg/gitwrite"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

// AttributeFunc has the signature of attrib.Attribute.
type AttributeFunc func(o cache.Options, repo gitx.Repo, head string, comps []component.Component, states map[string]component.State, globalGlobs []string) (map[string]attrib.Result, []string, error)

// Env is everything the release commands need.
type Env struct {
	// Repo is the Bazel workspace (possibly a subdirectory of the git
	// toplevel).
	Repo gitx.Repo
	// Forge may be nil in dry-run.
	Forge  forge.Forge
	Runner bazelq.Runner
	// Bazel is the bazel binary used by the default RunTarget.
	Bazel string
	// Remote is the git remote pushed to; MainBranch the branch release
	// PRs target. CommitURL is a commit URL template with {sha} (or "").
	Remote, MainBranch, CommitURL string
	// Cache configures attribution; its Repo and Runner default to Repo
	// and Runner. CacheDir also hosts release worktrees.
	Cache       cache.Options
	GlobalGlobs []string
	DryRun      bool
	// RetryTargets re-runs the release targets of components released by
	// HEAD even though their tag already exists.
	RetryTargets bool
	Now          func() time.Time
	Out, Warn    func(string)
	// RunTarget runs a release target with extra env entries in dir;
	// default BazelRunTarget(Bazel).
	RunTarget func(target string, env []string, dir string) error
	// Attribute defaults to attrib.Attribute.
	Attribute AttributeFunc
}

func (e Env) out(format string, a ...any) {
	if e.Out != nil {
		e.Out(fmt.Sprintf(format, a...))
	}
}

func (e Env) warn(format string, a ...any) {
	if e.Warn != nil {
		e.Warn(fmt.Sprintf(format, a...))
	}
}

func (e Env) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Plan is what a release of one component would do now.
type Plan struct {
	Component component.Component
	State     component.State
	Next      semver.Version
	// Release is true when attributed commits warrant releasing Next.
	Release bool
	// Entries are the attributed commits, newest first.
	Entries []changelog.Entry
	// Section is the changelog section of Next (empty unless Release).
	Section string
	// TagPending is true when VERSION at HEAD was set by a release commit
	// whose tag does not exist yet: `tag` has still to cut it. Attribution
	// then starts at that release commit.
	TagPending bool
}

// releaseSubject is the subject of the release commit of name at v.
func releaseSubject(name string, v semver.Version) string {
	return fmt.Sprintf("chore(release): %s %s", name, v)
}

// isReleaseOf reports whether subject is the release commit of name at v
// (a forge squash suffix " (#N)" is accepted).
func isReleaseOf(subject, name string, v semver.Version) bool {
	n, rv, ok := attrib.ParseReleaseSubject(subject)
	return ok && n == name && rv == v
}

// sortedComps returns comps sorted by name.
func sortedComps(comps []component.Component) []component.Component {
	out := append([]component.Component(nil), comps...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// pendingRelease returns the release commit of the component's VERSION at
// HEAD when that version is not tagged locally yet.
func pendingRelease(e Env, c component.Component, st component.State) (gitx.Commit, bool, error) {
	if st.Source != "version_file" || st.Current == nil {
		return gitx.Commit{}, false, nil
	}
	if _, ok, err := e.Repo.TagCommit(c.Tag(*st.Current)); err != nil || ok {
		return gitx.Commit{}, false, err
	}
	last, ok, err := gitwrite.LastCommitTouching(e.Repo, c.VersionFile)
	if err != nil || !ok || !isReleaseOf(last.Subject, c.Name, *st.Current) {
		return gitx.Commit{}, false, err
	}
	return last, true, nil
}

// Status computes the plan of every component, in name order. It writes
// nothing. A component whose state cannot be resolved is reported in the
// returned error and left out; the others are still planned.
func Status(e Env, comps []component.Component) ([]Plan, error) {
	comps = sortedComps(comps)
	tags, err := component.ListTags(e.Repo)
	if err != nil {
		return nil, err
	}
	var errs []error
	var ok []component.Component
	states := map[string]component.State{}
	pending := map[string]bool{}
	for _, c := range comps {
		st, warns, err := component.Current(e.Repo, c, tags)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name, err))
			continue
		}
		rc, isPending, err := pendingRelease(e, c, st)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name, err))
			continue
		}
		if isPending {
			// Current's "no tag" warning is expected here: tag is next.
			st.BaselineSHA = rc.SHA
			pending[c.Label] = true
		} else {
			for _, w := range warns {
				e.warn("%s", w)
			}
		}
		states[c.Label] = st
		ok = append(ok, c)
	}
	if len(ok) == 0 {
		return nil, errors.Join(errs...)
	}

	attribute := e.Attribute
	if attribute == nil {
		attribute = attrib.Attribute
	}
	o := e.Cache
	o.Warn = nil // warnings are returned and forwarded below
	if o.Repo.Dir == "" {
		o.Repo = e.Repo
	}
	if o.Runner == nil {
		o.Runner = e.Runner
	}
	results, warns, err := attribute(o, e.Repo, "HEAD", ok, states, e.GlobalGlobs)
	for _, w := range warns {
		e.warn("%s", w)
	}
	if err != nil {
		return nil, errors.Join(append(errs, err)...)
	}

	plans := make([]Plan, 0, len(ok))
	for _, c := range ok {
		st := states[c.Label]
		res := results[c.Label]
		p := Plan{Component: c, State: st, TagPending: pending[c.Label]}
		parsed := make([]conventional.Parsed, len(res.Commits))
		for i, cm := range res.Commits {
			parsed[i] = conventional.Parse(cm)
			p.Entries = append(p.Entries, changelog.Entry{Commit: cm, Parsed: parsed[i], Via: via(c, res.Files[cm.SHA])})
		}
		var nw []string
		p.Next, p.Release, nw = conventional.Next(st.Current, c.Initial, parsed)
		for _, w := range nw {
			e.warn("%s: %s", c.Name, w)
		}
		if p.Release {
			p.Section = changelog.Section(p.Next, e.now(), p.Entries, e.CommitURL, st.Current == nil)
		}
		plans = append(plans, p)
	}
	return plans, errors.Join(errs...)
}

// via returns the attributed files outside the component's package (nil
// for a root-package component).
func via(c component.Component, files []string) []string {
	if c.Package == "" {
		return nil
	}
	var out []string
	for _, f := range files {
		if !strings.HasPrefix(f, c.Package+"/") {
			out = append(out, f)
		}
	}
	return out
}

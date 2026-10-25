// Command release maintains release pull requests and cuts releases of
// release_component targets. Run it in CI with
//
//	bazel run //go/release -- <command> [flags]
//
// Commands: pr (open/update/close release PRs), tag (tag and publish merged
// releases), run (tag, then pr) and status (print the plan, write nothing).
//
// Exit codes: 0 success, 1 runtime error, 2 usage error.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/bazelq"
	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/cliutil"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/ignorefile"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
	"loopingz.com/rules_k8s_cd/release/pkg/attrib"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
	"loopingz.com/rules_k8s_cd/release/pkg/forge"
	"loopingz.com/rules_k8s_cd/release/pkg/release"
)

type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

// options are the parsed flags of a command.
type options struct {
	command                    string
	forge, forgeURL, repo      string
	remote, mainBranch         string
	components                 string
	dryRun, retryTargets       bool
	cacheDir, workspace, bazel string
	globalFiles, ignores       stringSlice
	ignoreFile                 string
	needsForge                 bool // the command writes to (or reads) the forge
}

// deps is what a command runs against.
type deps struct {
	Env release.Env
	// Discover lists the release components in a bazel query scope.
	Discover func(scope string) ([]component.Component, error)
}

// newEnv builds the dependencies of a command; tests replace it. warn and
// out are the sinks to wire into the release Env.
var newEnv = defaultNewEnv

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}

const usage = `usage: release <command> [flags]

commands:
  pr      open, update or close the release PR of every component
  tag     tag and publish releases whose release PR is merged
  run     tag, then pr (pr is skipped when tag fails)
  status  print the release plan of every component; writes nothing
`

func run(args []string, stdout, stderr io.Writer, env func(string) string) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd := args[0]
	switch cmd {
	case "-h", "-help", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	case "pr", "tag", "run", "status":
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n%s", cmd, usage)
		return 2
	}

	o := options{command: cmd}
	fs := flag.NewFlagSet("release "+cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.forge, "forge", "", "forge kind: github|forgejo (default: detected)")
	fs.StringVar(&o.forgeURL, "forge-url", "", "forge web URL (GitHub Enterprise or Forgejo)")
	fs.StringVar(&o.repo, "repo", "", "owner/name (default: from the remote URL)")
	fs.StringVar(&o.remote, "remote", "origin", "git remote to push to")
	fs.StringVar(&o.mainBranch, "main-branch", "", "branch release PRs target (default: the remote's HEAD branch, else main)")
	fs.StringVar(&o.components, "components", "//...", "bazel query scope searched for release_component targets")
	fs.BoolVar(&o.dryRun, "dry-run", false, "print what would be done; push, tag and write nothing")
	fs.BoolVar(&o.retryTargets, "retry-targets", false, "tag: finish the release of already published tags (forge release, release targets)")
	fs.StringVar(&o.cacheDir, "cache-dir", "", "cache directory (default <user cache>/rules_k8s_cd/release)")
	fs.StringVar(&o.workspace, "workspace", "", "workspace directory (default $BUILD_WORKSPACE_DIRECTORY, else cwd)")
	fs.StringVar(&o.bazel, "bazel", "bazel", "bazel binary")
	fs.Var(&o.globalFiles, "global-file", "extra glob of files affecting every component (repeatable)")
	fs.Var(&o.ignores, "ignore", "glob of files never attributed to any component (repeatable)")
	fs.StringVar(&o.ignoreFile, "ignore-file", ignorefile.DefaultName, "ignore file (commits per component, file globs), relative to the workspace; missing: none")
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	o.needsForge = cmd != "status"

	secrets := tokens(env)
	fail := func(err error) int {
		fmt.Fprintf(stderr, "error: %s\n", scrub(err.Error(), secrets))
		return 1
	}
	warn := func(msg string) { fmt.Fprintf(stderr, "warning: %s\n", scrub(msg, secrets)) }
	out := func(msg string) { fmt.Fprintln(stdout, scrub(msg, secrets)) }

	d, err := newEnv(o, env, out, warn)
	if err != nil {
		return fail(err)
	}
	comps, err := d.Discover(o.components)
	if err != nil {
		return fail(err)
	}
	if len(comps) == 0 {
		warn(fmt.Sprintf("no release_component found in %s", o.components))
	}

	e := d.Env
	switch cmd {
	case "status":
		plans, err := release.Status(e, comps)
		printStatus(stdout, plans)
		if err != nil {
			return fail(err)
		}
	case "tag":
		if err := release.Tag(e, comps); err != nil {
			return fail(err)
		}
	case "pr":
		if err := release.PR(e, comps); err != nil {
			return fail(err)
		}
	case "run":
		// A failed tag must not be followed by pr: pr could overwrite a
		// VERSION that was merged but never published.
		if err := release.Tag(e, comps); err != nil {
			code := fail(err)
			fmt.Fprintln(stderr, "error: tag failed; skipping pr")
			return code
		}
		if err := release.PR(e, comps); err != nil {
			return fail(err)
		}
	}
	return 0
}

// tokens returns the forge tokens set in the environment.
func tokens(env func(string) string) []string {
	var out []string
	for _, k := range []string{"FORGE_TOKEN", "GITHUB_TOKEN", "FORGEJO_TOKEN"} {
		if v := env(k); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// scrub masks every secret in s, a defence in depth: tokens are never
// formatted into messages on purpose.
func scrub(s string, secrets []string) string {
	for _, t := range secrets {
		s = strings.ReplaceAll(s, t, "***")
	}
	return s
}

// defaultNewEnv wires the real git, bazel and forge.
func defaultNewEnv(o options, env func(string) string, out, warn func(string)) (deps, error) {
	resolve := func(p string) string { return cliutil.ResolvePath(env, p) }

	ws := o.workspace
	if ws != "" {
		ws = resolve(ws)
	} else {
		ws = env("BUILD_WORKSPACE_DIRECTORY")
	}
	if ws == "" {
		wd, err := os.Getwd()
		if err != nil {
			return deps{}, err
		}
		ws = wd
	}
	ws, err := filepath.Abs(ws)
	if err != nil {
		return deps{}, err
	}
	repo := gitx.Repo{Dir: ws}
	if _, err := repo.RevParse("HEAD"); err != nil {
		return deps{}, fmt.Errorf("resolving HEAD: %w", err)
	}
	ign, iw, err := ignorefile.Load(ws, o.ignoreFile)
	if err != nil {
		return deps{}, fmt.Errorf("--ignore-file: %w", err)
	}
	for _, w := range iw {
		warn(w)
	}
	ignores := append(append([]string(nil), o.ignores...), ign.Paths...)
	if o.ignoreFile != "" && !filepath.IsAbs(o.ignoreFile) {
		ignores = append(ignores, filepath.ToSlash(filepath.Clean(o.ignoreFile)))
	}

	// Forge. Dry-run never needs one: release tolerates a nil Forge and
	// then only prints what it would do; with a token the forge is still
	// consulted read-only (FindOpenPR, ReleaseExists) for a precise preview.
	var fg forge.Forge
	if o.needsForge {
		fg, err = newForge(o, env, repo, warn)
		if err != nil {
			return deps{}, err
		}
	}

	mainBranch := o.mainBranch
	if mainBranch == "" {
		mainBranch = defaultMainBranch(repo, o.remote)
	}

	cd := o.cacheDir
	if cd != "" {
		cd = resolve(cd)
	} else {
		cd = cliutil.DefaultCacheDir(env, "release")
	}
	if dirty, err := repo.Dirty(); err != nil {
		warn(fmt.Sprintf("cannot check working tree state: %v; cache disabled for this run", err))
		cd = ""
	} else if dirty {
		warn("working tree has uncommitted changes; cache disabled for this run")
		cd = ""
	}

	runner := bazelq.ExecRunner{Bazel: o.bazel, Dir: ws}
	e := release.Env{
		Repo:         repo,
		Forge:        fg,
		Runner:       runner,
		Bazel:        o.bazel,
		Remote:       o.remote,
		MainBranch:   mainBranch,
		CommitURL:    repo.CommitURLTemplate(),
		Cache:        cache.Options{Workspace: ws, CacheDir: cd, Repo: repo, Runner: runner},
		GlobalGlobs:  o.globalFiles,
		DryRun:       o.dryRun,
		RetryTargets: o.retryTargets,
		Out:          out,
		Warn:         warn,
		RunTarget:    release.BazelRunTarget(o.bazel),
		Attribute:    ignoringCommits(ignoring(attrib.Attribute, ignores), ign, repo.HasCommit, warn),
	}
	return deps{
		Env:      e,
		Discover: func(scope string) ([]component.Component, error) { return component.Discover(runner, scope) },
	}, nil
}

// newForge builds the forge client. Without a token it is an error, except
// in dry-run where it returns a nil forge (and a failed detection is only
// a warning).
func newForge(o options, env func(string) string, repo gitx.Repo, warn func(string)) (forge.Forge, error) {
	remoteURL := ""
	if b, err := repo.Git("remote", "get-url", o.remote); err == nil {
		remoteURL = strings.TrimSpace(string(b))
	}
	cfg, err := forge.Detect(env, remoteURL, forge.Config{Kind: o.forge, URL: o.forgeURL, Repo: o.repo})
	if err != nil {
		if o.dryRun {
			warn(fmt.Sprintf("%v; dry-run continues without forge lookups", err))
			return nil, nil
		}
		return nil, err
	}
	if cfg.Token == "" {
		if o.dryRun {
			warn("no forge token (FORGE_TOKEN, GITHUB_TOKEN or FORGEJO_TOKEN); dry-run continues without forge lookups")
			return nil, nil
		}
		return nil, errors.New("no forge token: set FORGE_TOKEN, GITHUB_TOKEN or FORGEJO_TOKEN")
	}
	if cfg.Kind == "forgejo" {
		return forge.NewForgejo(cfg.URL, cfg.Owner, cfg.Repo, cfg.Token, http.DefaultClient), nil
	}
	return forge.NewGitHub(cfg.URL, cfg.Owner, cfg.Repo, cfg.Token, http.DefaultClient), nil
}

// defaultMainBranch is the branch <remote>/HEAD points to, else "main".
func defaultMainBranch(repo gitx.Repo, remote string) string {
	b, err := repo.Git("symbolic-ref", "--short", "refs/remotes/"+remote+"/HEAD")
	if err == nil {
		if br := strings.TrimPrefix(strings.TrimSpace(string(b)), remote+"/"); br != "" {
			return br
		}
	}
	return "main"
}

// ignoringCommits wraps an attribution so the commits the ignore file (or a
// Release-Ignore trailer) names for a component, or one of its targets, are
// never attributed to it. Bad entries only warn, once.
func ignoringCommits(inner release.AttributeFunc, ign ignorefile.File, exists func(string) bool, warn func(string)) release.AttributeFunc {
	checked := false
	return func(o cache.Options, repo gitx.Repo, head string, comps []component.Component, states map[string]component.State, globalGlobs []string) (map[string]attrib.Result, []string, error) {
		res, warns, err := inner(o, repo, head, comps, states, globalGlobs)
		if err != nil {
			return res, warns, err
		}
		if !checked {
			checked = true
			names := map[string]bool{}
			targets := map[string]bool{}
			for _, r := range res {
				names[r.Component.Name] = true
				for _, t := range r.Targets {
					targets[t] = true
				}
			}
			for _, c := range comps {
				names[c.Name] = true
			}
			for _, w := range ign.Check(exists, false, func(n string) bool { return names[n] }) {
				warn(w)
			}
			for _, w := range ign.Check(nil, true, func(l string) bool { return targets[l] }) {
				warn(w)
			}
		}
		for label, r := range res {
			keys := append([]string{r.Component.Name}, r.Targets...)
			var commits []gitx.Commit
			for _, c := range r.Commits {
				if ign.Ignores(c.SHA, c.Body, keys...) {
					delete(r.Files, c.SHA)
					continue
				}
				commits = append(commits, c)
			}
			r.Commits = commits
			res[label] = r
		}
		return res, warns, nil
	}
}

// ignoring wraps an attribution so files matching globs never count: a
// commit whose attributing files are all ignored is dropped.
func ignoring(inner release.AttributeFunc, globs []string) release.AttributeFunc {
	if len(globs) == 0 {
		return inner
	}
	ignored := func(p string) bool {
		for _, g := range globs {
			if report.MatchGlob(g, p) {
				return true
			}
		}
		return false
	}
	return func(o cache.Options, repo gitx.Repo, head string, comps []component.Component, states map[string]component.State, globalGlobs []string) (map[string]attrib.Result, []string, error) {
		res, warns, err := inner(o, repo, head, comps, states, globalGlobs)
		if err != nil {
			return res, warns, err
		}
		for label, r := range res {
			var commits []gitx.Commit
			files := map[string][]string{}
			for _, c := range r.Commits {
				var keep []string
				for _, f := range r.Files[c.SHA] {
					if !ignored(f) {
						keep = append(keep, f)
					}
				}
				if len(keep) == 0 && len(r.Files[c.SHA]) > 0 {
					continue
				}
				commits = append(commits, c)
				files[c.SHA] = keep
			}
			r.Commits, r.Files = commits, files
			res[label] = r
		}
		return res, warns, nil
	}
}

// printStatus writes the plans as a table sorted by component name.
func printStatus(w io.Writer, plans []release.Plan) {
	if len(plans) == 0 {
		return
	}
	plans = append([]release.Plan(nil), plans...)
	sort.SliceStable(plans, func(i, j int) bool { return plans[i].Component.Name < plans[j].Component.Name })
	rows := [][4]string{{"component", "current", "next", "state"}}
	for _, p := range plans {
		cur, next := "-", "-"
		if p.State.Current != nil {
			cur = p.State.Current.String()
		}
		if p.Release {
			next = p.Next.String()
		}
		rows = append(rows, [4]string{p.Component.Name, cur, next, stateOf(p)})
	}
	var width [4]int
	for _, r := range rows {
		for i, c := range r {
			if len(c) > width[i] {
				width[i] = len(c)
			}
		}
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%-*s | %-*s | %-*s | %s\n", width[0], r[0], width[1], r[1], width[2], r[2], r[3])
	}
}

// stateOf classifies a plan: a merged but untagged release awaits `tag`
// (pr leaves it alone), else a releasable component has or gets a release PR.
func stateOf(p release.Plan) string {
	switch {
	case p.TagPending:
		return "awaiting tag"
	case p.Release && p.State.Current == nil:
		return "first release"
	case p.Release:
		return "release PR"
	}
	return "up to date"
}

package release

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"loopingz.com/rules_k8s_cd/release/pkg/changelog"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
	"loopingz.com/rules_k8s_cd/release/pkg/gitwrite"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

// Tag cuts the release of every component, in name order, whose VERSION at
// HEAD is v, whose tag for v is not on the remote yet, and whose last commit
// touching VERSION is the release commit of v: it tags that commit, pushes
// the tag, creates the forge release (notes: the changelog section of v at
// the release commit) and runs the release targets.
//
// An existing tag is never created again. The forge release and the release
// targets follow only a tag published in this run; with RetryTargets they
// also follow the already published tag of the current VERSION, provided it
// points at that version's release commit (the forge release is then created
// if missing and every target re-runs). A failure after tagging is reported
// with a warning to rerun with --retry-targets. Errors of one component do
// not stop the others.
func Tag(e Env, comps []component.Component) error {
	if e.Remote == "" {
		return errors.New("release tag: remote is required")
	}
	if e.Forge == nil && !e.DryRun {
		return errors.New("release tag: no forge configured")
	}
	var errs []error
	for _, c := range sortedComps(comps) {
		if err := tagOne(e, c); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name, err))
		}
	}
	return errors.Join(errs...)
}

func tagOne(e Env, c component.Component) error {
	if c.VersionFile == "" {
		return nil
	}
	b, ok, err := e.Repo.Show("HEAD", c.VersionFile)
	if err != nil || !ok {
		return err
	}
	v, err := semver.Parse(strings.TrimSpace(string(b)))
	if err != nil {
		return fmt.Errorf("%s: %w", c.VersionFile, err)
	}
	tag := c.Tag(v)

	sha, published, err := gitwrite.RemoteTagCommit(e.Repo, e.Remote, tag)
	if err != nil {
		return err
	}
	if published && !e.RetryTargets {
		return nil
	}
	last, ok, err := gitwrite.LastCommitTouching(e.Repo, c.VersionFile)
	if err != nil {
		return err
	}
	isRelease := ok && isReleaseOf(last.Subject, c.Name, v)
	if published {
		if !isRelease || last.SHA != sha {
			e.out("%s: %s is not at the release commit of %s; nothing to retry", c.Name, tag, c.VersionFile)
			return nil
		}
	} else {
		if !isRelease {
			e.warn("VERSION of %s changed outside a release commit; not releasing", c.Name)
			return nil
		}
		sha = last.SHA
		if e.DryRun {
			e.out("%s: would tag %s at %s", c.Name, tag, last.Short)
		} else {
			if err := gitwrite.CreateTag(e.Repo, e.Remote, tag, sha, "Release "+tag); err != nil {
				return err
			}
			e.out("%s: tagged %s at %s", c.Name, tag, last.Short)
		}
	}

	if err := ensureRelease(e, c, tag, sha, v); err != nil {
		e.warn("%s: %s was tagged but release failed; rerun with --retry-targets", c.Name, tag)
		return err
	}
	if len(c.ReleaseTargets) == 0 {
		return nil
	}
	if e.DryRun {
		for _, t := range c.ReleaseTargets {
			e.out("%s: would run %s for %s", c.Name, t, tag)
		}
		return nil
	}
	return runTargets(e, c, v, tag, sha)
}

// ensureRelease creates the forge release of tag unless it exists.
func ensureRelease(e Env, c component.Component, tag, sha string, v semver.Version) error {
	if e.DryRun && e.Forge == nil {
		e.out("%s: would create the forge release %s unless it exists", c.Name, tag)
		return nil
	}
	exists, err := e.Forge.ReleaseExists(tag)
	if err != nil || exists {
		return err
	}
	notes := ""
	if c.Changelog != "" {
		content, _, err := e.Repo.Show(sha, c.Changelog)
		if err != nil {
			return err
		}
		notes = changelog.Extract(content, v)
	}
	if notes == "" {
		notes = "Release " + tag
	}
	if e.DryRun {
		e.out("%s: would create the forge release %s", c.Name, tag)
		return nil
	}
	if err := e.Forge.CreateRelease(tag, tag, notes); err != nil {
		return err
	}
	e.out("%s: created the forge release %s", c.Name, tag)
	return nil
}

// runTargets runs the release targets in order, in the workspace when HEAD
// is the release commit, else in a temporary worktree of it.
func runTargets(e Env, c component.Component, v semver.Version, tag, sha string) (err error) {
	head, err := e.Repo.RevParse("HEAD")
	if err != nil {
		return err
	}
	dir := e.Repo.Dir
	if head != sha {
		wt, cleanup, err := worktree(e, sha)
		if err != nil {
			return err
		}
		defer cleanup()
		dir = wt
	}
	env := []string{"RELEASE_COMPONENT=" + c.Name, "RELEASE_VERSION=" + v.String(), "RELEASE_TAG=" + tag}
	run := e.RunTarget
	if run == nil {
		run = BazelRunTarget(e.Bazel)
	}
	for _, t := range c.ReleaseTargets {
		e.out("%s: running %s for %s", c.Name, t, tag)
		if err := run(t, env, dir); err != nil {
			e.warn("%s: %s was tagged but target %s failed; rerun with --retry-targets", c.Name, tag, t)
			return fmt.Errorf("release target %s: %w", t, err)
		}
	}
	return nil
}

// worktree checks sha out in a detached worktree under the cache dir (or
// the temp dir) and returns the workspace directory inside it.
func worktree(e Env, sha string) (string, func(), error) {
	base := e.Cache.CacheDir
	if base == "" {
		base = os.TempDir()
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp(base, "release-worktree-")
	if err != nil {
		return "", nil, err
	}
	wt := filepath.Join(tmp, "wt")
	cleanup := func() {
		if _, err := e.Repo.Git("worktree", "remove", "--force", wt); err != nil {
			e.warn("removing worktree %s: %v", wt, err)
		}
		os.RemoveAll(tmp)
		// Best effort: a stale registration is harmless and pruned later.
		_, _ = e.Repo.Git("worktree", "prune")
	}
	prefix, err := e.Repo.Git("rev-parse", "--show-prefix")
	if err != nil {
		os.RemoveAll(tmp)
		return "", nil, err
	}
	if _, err := e.Repo.Git("-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", wt, sha); err != nil {
		cleanup()
		return "", nil, err
	}
	return filepath.Join(wt, strings.TrimSpace(string(prefix))), cleanup, nil
}

// BazelRunTarget returns a RunTarget that runs
// `bazel run --stamp --embed_label=$RELEASE_TAG <target>` in dir with env
// added to the process environment; output goes to stderr.
func BazelRunTarget(bazel string) func(target string, env []string, dir string) error {
	if bazel == "" {
		bazel = "bazel"
	}
	return func(target string, env []string, dir string) error {
		tag := ""
		for _, kv := range env {
			if strings.HasPrefix(kv, "RELEASE_TAG=") {
				tag = strings.TrimPrefix(kv, "RELEASE_TAG=")
			}
		}
		cmd := exec.Command(bazel, "run", "--stamp", "--embed_label="+tag, target)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), env...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

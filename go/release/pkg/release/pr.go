package release

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/release/pkg/changelog"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
	"loopingz.com/rules_k8s_cd/release/pkg/forge"
	"loopingz.com/rules_k8s_cd/release/pkg/gitwrite"
)

// PR opens, updates or closes the release PR of every component, in name
// order. A releasable component gets branch release/<name> force-pushed
// with one release commit on top of HEAD (VERSION, changelog, bump files)
// and an open PR for it; a component with nothing to release has its open
// release PR closed. A component whose release is merged but not tagged yet
// is left alone, so the untagged version is never overwritten. HEAD must be
// the tip of the main branch; when main moved on since the checkout, pr is
// skipped with a warning (see checkMainTip). Errors of one component do not stop the
// others; they are joined.
func PR(e Env, comps []component.Component) error {
	if e.Remote == "" || e.MainBranch == "" {
		return errors.New("release pr: remote and main branch are required")
	}
	if e.Forge == nil && !e.DryRun {
		return errors.New("release pr: no forge configured")
	}
	if skip, err := checkMainTip(e); err != nil || skip {
		return err
	}
	plans, err := Status(e, comps)
	errs := []error{err}
	for _, p := range plans {
		if err := prOne(e, p); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Component.Name, err))
		}
	}
	return errors.Join(errs...)
}

// checkMainTip errors when HEAD is not the tip of <remote>/<main> and has
// diverged from it. When main only moved on since the checkout (HEAD is an
// ancestor of the tip) it warns and returns skip: the run triggered by the
// newer commits updates the PRs. It only warns when the remote-tracking ref
// is unknown.
func checkMainTip(e Env) (skip bool, err error) {
	head, err := e.Repo.RevParse("HEAD")
	if err != nil {
		return false, err
	}
	ref := "refs/remotes/" + e.Remote + "/" + e.MainBranch
	out, err := e.Repo.Git("rev-parse", "--verify", "--quiet", "--end-of-options", ref+"^{commit}")
	if err != nil {
		e.warn("cannot check that HEAD is the tip of %s/%s: %s not found", e.Remote, e.MainBranch, ref)
		return false, nil
	}
	tip := strings.TrimSpace(string(out))
	if tip == head {
		return false, nil
	}
	if _, err := e.Repo.Git("merge-base", "--is-ancestor", head, tip); err == nil {
		e.warn("%s moved since checkout; skipping pr (the next run will update PRs)", e.MainBranch)
		return true, nil
	}
	return false, fmt.Errorf("release pr: HEAD %s is not the tip of %s/%s (%s); check out the main branch", head, e.Remote, e.MainBranch, tip)
}

func prOne(e Env, p Plan) error {
	c := p.Component
	branch := "release/" + c.Name
	if p.TagPending {
		e.out("%s: %s is merged but not tagged yet; run tag first", c.Name, p.State.Current)
		return nil
	}
	if !p.Release {
		return closeStale(e, c.Name, branch)
	}
	files, err := releaseFiles(e, p)
	if err != nil {
		return err
	}
	title := releaseSubject(c.Name, p.Next)
	base, err := e.Repo.RevParse("HEAD")
	if err != nil {
		return err
	}
	// The trailer, in the release commit and in the PR body (forges may use
	// either as the squash commit message), lets tag spot commits that
	// landed on main after this run and are missing from the changelog.
	trailer := releaseBaseTrailer + base
	body := strings.TrimRight(changelog.PRBody(c.Name, p.Section), "\n") + "\n\n" + trailer + "\n"

	if e.DryRun {
		names := make([]string, 0, len(files))
		for n := range files {
			names = append(names, n)
		}
		sort.Strings(names)
		e.out("%s: would push %s with %q changing %s", c.Name, branch, title, strings.Join(names, ", "))
		if e.Forge == nil {
			e.out("%s: would open or update the release PR %q", c.Name, title)
			return nil
		}
		pr, err := findReleasePR(e, c.Name, branch)
		if err != nil {
			return err
		}
		if pr != nil {
			e.out("%s: would update PR #%d to %q", c.Name, pr.Number, title)
		} else {
			e.out("%s: would open PR %q", c.Name, title)
		}
		return nil
	}

	pr, err := findReleasePR(e, c.Name, branch)
	if err != nil {
		return err
	}
	// Rebuilding an unchanged release PR on every push to main only re-runs
	// its checks: keep it unless the component tracks the main tip.
	if pr != nil && !c.TrackMainTip {
		same, err := branchHasFiles(e, branch, title, files)
		if err != nil {
			return err
		}
		if same {
			e.out("%s: PR #%d %q is up to date", c.Name, pr.Number, title)
			return nil
		}
	}
	sha, err := gitwrite.CommitFiles(e.Repo, base, files, title+"\n\n"+trailer)
	if err != nil {
		return err
	}
	if err := gitwrite.PushBranch(e.Repo, e.Remote, sha, branch); err != nil {
		return err
	}
	if pr != nil {
		if err := e.Forge.UpdatePR(pr.Number, title, body); err != nil {
			return err
		}
		e.out("%s: updated PR #%d %q", c.Name, pr.Number, title)
		return nil
	}
	pr, err = e.Forge.CreatePR(branch, e.MainBranch, title, body)
	if err != nil {
		return err
	}
	e.out("%s: opened PR #%d %q %s", c.Name, pr.Number, title, pr.URL)
	return nil
}

// branchHasFiles reports whether <remote>/<branch> is one release commit
// titled title, on a commit of main's history (an ancestor of HEAD), that
// changes only files and gives each of them its content in files. A missing
// branch is not up to date.
func branchHasFiles(e Env, branch, title string, files map[string][]byte) (bool, error) {
	out, err := e.Repo.Git("ls-remote", "--heads", e.Remote, "refs/heads/"+branch)
	if err != nil {
		return false, err
	}
	sha, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\t")
	if sha == "" {
		return false, nil
	}
	if !e.Repo.HasCommit(sha) {
		if _, err := e.Repo.Git("fetch", "--quiet", "--no-tags", e.Remote, "refs/heads/"+branch); err != nil {
			return false, err
		}
	}
	out, err = e.Repo.Git("log", "-1", "--format=%s%x00%P", sha)
	if err != nil {
		return false, err
	}
	subject, parents, _ := strings.Cut(strings.TrimSpace(string(out)), "\x00")
	if subject != title || strings.Contains(parents, " ") || parents == "" {
		return false, nil
	}
	if onMain, err := e.Repo.IsAncestor(parents, "HEAD"); err != nil || !onMain {
		return false, err
	}
	out, err = e.Repo.Git("diff", "--no-renames", "--name-only", "-z", parents, sha)
	if err != nil {
		return false, err
	}
	for _, name := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if _, ok := files[name]; !ok {
			return false, nil
		}
	}
	for name, want := range files {
		got, ok, err := e.Repo.Show(sha, name)
		if err != nil || !ok || !bytes.Equal(got, want) {
			return false, err
		}
	}
	return true, nil
}

// findReleasePR returns the open release PR of component name on branch:
// an open PR on branch whose body carries the release marker. Another PR on
// branch (opened by hand, or from a fork on forges matching the branch name
// only) is never updated or closed: it is reported and nil is returned, so
// creating the release PR then fails loudly if the forge refuses a second
// PR for the branch.
func findReleasePR(e Env, name, branch string) (*forge.PR, error) {
	pr, err := e.Forge.FindOpenPR(branch)
	if err != nil || pr == nil {
		return nil, err
	}
	if !strings.Contains(pr.Body, changelog.Marker(name)) {
		e.warn("%s: PR #%d on %s lacks the release marker %q; leaving it untouched", name, pr.Number, branch, changelog.Marker(name))
		return nil, nil
	}
	return pr, nil
}

// closeStale closes the open release PR of a component with nothing to
// release.
func closeStale(e Env, name, branch string) error {
	if e.Forge == nil {
		e.out("%s: nothing to release; would close its open release PR, if any", name)
		return nil
	}
	pr, err := findReleasePR(e, name, branch)
	if err != nil || pr == nil {
		return err
	}
	if e.DryRun {
		e.out("%s: nothing to release; would close PR #%d", name, pr.Number)
		return nil
	}
	if err := e.Forge.ClosePR(pr.Number, fmt.Sprintf("No releasable changes for %s anymore.", name)); err != nil {
		return err
	}
	e.out("%s: nothing to release; closed PR #%d", name, pr.Number)
	return nil
}

// releaseFiles returns the files of the release commit of p.
func releaseFiles(e Env, p Plan) (map[string][]byte, error) {
	c := p.Component
	if c.VersionFile == "" {
		return nil, errors.New("no version_file to write the release version to")
	}
	next := p.Next.String()
	files := map[string][]byte{c.VersionFile: []byte(next + "\n")}
	if c.Changelog != "" {
		cur, _, err := e.Repo.Show("HEAD", c.Changelog)
		if err != nil {
			return nil, err
		}
		files[c.Changelog] = changelog.Prepend(cur, p.Section)
	}
	paths := make([]string, 0, len(c.BumpFiles))
	for path := range c.BumpFiles {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if _, dup := files[path]; dup {
			return nil, fmt.Errorf("bump_files %s is also the version file or changelog", path)
		}
		re, err := regexp.Compile(c.BumpFiles[path])
		if err != nil {
			return nil, fmt.Errorf("bump_files %s: %w", path, err)
		}
		if re.NumSubexp() != 1 {
			return nil, fmt.Errorf("bump_files %s: pattern %q must have exactly one capturing group", path, c.BumpFiles[path])
		}
		cur, ok, err := e.Repo.Show("HEAD", path)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("bump_files %s does not exist", path)
		}
		out, n := replaceGroup(re, cur, next)
		if n == 0 {
			return nil, fmt.Errorf("bump_files %s: pattern %q matches nothing", path, c.BumpFiles[path])
		}
		files[path] = out
	}
	return files, nil
}

// replaceGroup replaces the first capturing group of every match of re in
// b with repl; n counts the replacements.
func replaceGroup(re *regexp.Regexp, b []byte, repl string) (out []byte, n int) {
	last := 0
	for _, m := range re.FindAllSubmatchIndex(b, -1) {
		if m[2] < 0 {
			continue
		}
		out = append(out, b[last:m[2]]...)
		out = append(out, repl...)
		last = m[3]
		n++
	}
	return append(out, b[last:]...), n
}

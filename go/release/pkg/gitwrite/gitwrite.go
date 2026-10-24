// Package gitwrite performs the release tool's git write operations with
// plumbing only: it never touches the checkout's working tree, index or HEAD.
package gitwrite

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
)

// identityEnv returns author/committer env entries only when git cannot
// resolve a committer identity on its own.
func identityEnv(repo gitx.Repo) []string {
	if _, err := repo.Git("var", "GIT_COMMITTER_IDENT"); err == nil {
		return nil
	}
	return []string{
		"GIT_AUTHOR_NAME=rules_k8s_cd release", "GIT_AUTHOR_EMAIL=release@localhost",
		"GIT_COMMITTER_NAME=rules_k8s_cd release", "GIT_COMMITTER_EMAIL=release@localhost",
	}
}

// CommitFiles creates (without checking out) a commit on top of parent whose
// tree is parent's tree plus files. Keys of files are relative to repo.Dir;
// parent is a revision. It returns the new commit SHA. The working tree, the
// real index and HEAD are untouched.
func CommitFiles(repo gitx.Repo, parent string, files map[string][]byte, message string) (string, error) {
	out, err := repo.Git("rev-parse", "--show-prefix")
	if err != nil {
		return "", err
	}
	prefix := strings.TrimSpace(string(out))

	parentSHA, err := repo.RevParse(parent)
	if err != nil {
		return "", err
	}

	names := make([]string, 0, len(files))
	seen := map[string]string{}
	for name := range files {
		clean := path.Clean(filepath.ToSlash(name))
		if name == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || clean == "." {
			return "", fmt.Errorf("gitwrite: invalid path %q", name)
		}
		if other, dup := seen[clean]; dup {
			return "", fmt.Errorf("gitwrite: paths %q and %q are the same file", other, name)
		}
		seen[clean] = name
		names = append(names, name)
	}
	sort.Strings(names)

	tmp, err := os.MkdirTemp("", "gitwrite-index-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(tmp, "index")}

	if _, err := repo.GitEnv(env, "read-tree", parentSHA); err != nil {
		return "", err
	}
	for _, name := range names {
		blob, err := hashObject(repo, files[name])
		if err != nil {
			return "", err
		}
		p := prefix + path.Clean(filepath.ToSlash(name))
		if _, err := repo.GitEnv(env, "update-index", "--add", "--cacheinfo", "100644,"+blob+","+p); err != nil {
			return "", err
		}
	}
	tree, err := repo.GitEnv(env, "write-tree")
	if err != nil {
		return "", err
	}
	cenv := append(env, identityEnv(repo)...)
	// commit-tree honours commit.gpgsign, so override hostile local config.
	sha, err := repo.GitEnv(cenv, "-c", "commit.gpgsign=false", "commit-tree",
		strings.TrimSpace(string(tree)), "-p", parentSHA, "-m", message)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(sha)), nil
}

func hashObject(repo gitx.Repo, content []byte) (string, error) {
	out, err := repo.GitEnvStdin(nil, content, "hash-object", "-w", "--stdin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// checkArg rejects empty values and values git could parse as an option.
func checkArg(what, v string) error {
	if v == "" || strings.HasPrefix(v, "-") || strings.HasPrefix(v, "+") {
		return fmt.Errorf("gitwrite: invalid %s %q", what, v)
	}
	return nil
}

// PushBranch force-pushes commit sha to refs/heads/<branch> on remote. Only
// branches under release/ may be force-pushed.
func PushBranch(repo gitx.Repo, remote, sha, branch string) error {
	for _, c := range [][2]string{{"remote", remote}, {"sha", sha}, {"branch", branch}} {
		if err := checkArg(c[0], c[1]); err != nil {
			return err
		}
	}
	if !strings.HasPrefix(branch, "release/") || strings.ContainsAny(branch, ":") {
		return fmt.Errorf("gitwrite: refusing to force-push branch %q outside release/", branch)
	}
	if _, err := repo.Git("check-ref-format", "--branch", branch); err != nil {
		return fmt.Errorf("gitwrite: invalid branch %q: %w", branch, err)
	}
	_, err := repo.Git("-c", "core.hooksPath=/dev/null", "push", "--force", remote, sha+":refs/heads/"+branch)
	return err
}

// RemoteTagCommit returns the commit tag points to on remote (peeling
// annotated tags), if present there.
func RemoteTagCommit(repo gitx.Repo, remote, tag string) (string, bool, error) {
	for _, c := range [][2]string{{"remote", remote}, {"tag", tag}} {
		if err := checkArg(c[0], c[1]); err != nil {
			return "", false, err
		}
	}
	out, err := repo.Git("ls-remote", "--tags", remote, "refs/tags/"+tag, "refs/tags/"+tag+"^{}")
	if err != nil {
		return "", false, err
	}
	var direct, peeled string
	for _, l := range strings.Split(string(out), "\n") {
		f := strings.Fields(l)
		if len(f) != 2 {
			continue
		}
		switch f[1] {
		case "refs/tags/" + tag:
			direct = f[0]
		case "refs/tags/" + tag + "^{}":
			peeled = f[0]
		}
	}
	if peeled != "" {
		return peeled, true, nil
	}
	return direct, direct != "", nil
}

// CreateTag creates annotated tag at sha (a no-op if it already points
// there, locally or on remote) and pushes it to remote. A tag that exists at
// another commit, locally or on remote, is an error.
func CreateTag(repo gitx.Repo, remote, tag, sha, message string) error {
	for _, c := range [][2]string{{"remote", remote}, {"sha", sha}, {"tag", tag}} {
		if err := checkArg(c[0], c[1]); err != nil {
			return err
		}
	}
	if _, err := repo.Git("check-ref-format", "refs/tags/"+tag); err != nil {
		return fmt.Errorf("gitwrite: invalid tag %q: %w", tag, err)
	}
	want, err := repo.RevParse(sha)
	if err != nil {
		return err
	}
	if rc, ok, err := RemoteTagCommit(repo, remote, tag); err != nil {
		return err
	} else if ok {
		if rc != want {
			return fmt.Errorf("gitwrite: tag %s already exists on %s at %s, refusing to move it to %s", tag, remote, rc, want)
		}
		return nil
	}
	have, exists, err := repo.TagCommit(tag)
	if err != nil {
		return err
	}
	switch {
	case exists && have != want:
		return fmt.Errorf("gitwrite: tag %s already exists at %s, refusing to move it to %s", tag, have, want)
	case !exists:
		if _, err := repo.GitEnv(identityEnv(repo), "-c", "tag.gpgSign=false", "tag", "-a", tag, want, "-m", message); err != nil {
			return err
		}
	}
	_, err = repo.Git("-c", "core.hooksPath=/dev/null", "push", remote, "refs/tags/"+tag)
	return err
}

// LastCommitTouching returns the most recent commit that touched path
// (relative to repo.Dir); false if none did.
func LastCommitTouching(repo gitx.Repo, path string) (gitx.Commit, bool, error) {
	out, err := repo.Git("--literal-pathspecs", "log", "-1", "--format=%H", "--", path)
	if err != nil {
		return gitx.Commit{}, false, err
	}
	sha := strings.TrimSpace(string(out))
	if sha == "" {
		return gitx.Commit{}, false, nil
	}
	cs, err := repo.Log("", sha+"^!")
	if err != nil {
		return gitx.Commit{}, false, err
	}
	if len(cs) != 1 {
		return gitx.Commit{}, false, fmt.Errorf("gitwrite: expected one commit for %s, got %d", sha, len(cs))
	}
	return cs[0], true, nil
}

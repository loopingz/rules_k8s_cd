// Package gitwrite performs the release tool's git write operations with
// plumbing only: it never touches the checkout's working tree, index or HEAD.
package gitwrite

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
	for name := range files {
		clean := path.Clean(filepath.ToSlash(name))
		if name == "" || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || clean == "." {
			return "", fmt.Errorf("gitwrite: invalid path %q", name)
		}
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
	sha, err := repo.GitEnv(cenv, "-c", "commit.gpgsign=false", "commit-tree",
		strings.TrimSpace(string(tree)), "-p", parentSHA, "-m", message)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(sha)), nil
}

func hashObject(repo gitx.Repo, content []byte) (string, error) {
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Dir = repo.Dir
	cmd.Stdin = bytes.NewReader(content)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gitwrite: git hash-object: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// PushBranch force-pushes commit sha to refs/heads/<branch> on remote.
func PushBranch(repo gitx.Repo, remote, sha, branch string) error {
	_, err := repo.Git("-c", "core.hooksPath=/dev/null", "push", "--force", remote, sha+":refs/heads/"+branch)
	return err
}

// CreateTag creates annotated tag at sha (a no-op if it already points
// there) and pushes it to remote. A tag that exists at another commit is an
// error.
func CreateTag(repo gitx.Repo, remote, tag, sha, message string) error {
	if strings.HasPrefix(tag, "-") {
		return fmt.Errorf("gitwrite: invalid tag name %q", tag)
	}
	want, err := repo.RevParse(sha)
	if err != nil {
		return err
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
	out, err := repo.Git("log", "-1", "--format=%H", "--", path)
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

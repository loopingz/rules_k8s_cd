// Package gitx is the change_report CLI's only interface to git. It shells
// out to the git binary.
package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Repo is a git repository working directory. Dir may be a subdirectory of
// the git toplevel (e.g. a Bazel workspace inside a monorepo). All paths
// accepted and returned by Repo methods are relative to Dir; changes outside
// Dir are not reported.
type Repo struct {
	Dir string
}

// run executes git with args in r.Dir and returns stdout. On failure the
// error includes stderr; a non-zero exit is returned as *exec.ExitError
// (wrapped) so callers can inspect the code.
func (r Repo) run(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, &runError{args: args, err: err, stderr: strings.TrimSpace(stderr.String())}
	}
	return stdout.Bytes(), nil
}

type runError struct {
	args   []string
	err    error
	stderr string
}

func (e *runError) Error() string {
	return fmt.Sprintf("git %s: %v: %s", strings.Join(e.args, " "), e.err, e.stderr)
}

func (e *runError) Unwrap() error { return e.err }

func exitCode(err error) int {
	if ee, ok := err.(*runError); ok {
		if x, ok := ee.err.(*exec.ExitError); ok {
			return x.ExitCode()
		}
	}
	return -1
}

// RevParse resolves rev to a full commit SHA.
func (r Repo) RevParse(rev string) (string, error) {
	out, err := r.run("rev-parse", "--verify", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsAncestor reports whether a is an ancestor of (or equal to) b.
func (r Repo) IsAncestor(a, b string) (bool, error) {
	_, err := r.run("merge-base", "--is-ancestor", a, b)
	if err == nil {
		return true, nil
	}
	if exitCode(err) == 1 {
		return false, nil
	}
	return false, err
}

// RevSet returns the set of commit SHAs in from..to.
func (r Repo) RevSet(from, to string) (map[string]bool, error) {
	out, err := r.run("rev-list", rangeSpec(from, to))
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, l := range strings.Fields(string(out)) {
		set[l] = true
	}
	return set, nil
}

func rangeSpec(from, to string) string {
	if from == "" {
		return to
	}
	return from + ".." + to
}

// TreeFiles maps every file path in rev's tree to its blob SHA.
func (r Repo) TreeFiles(rev string) (map[string]string, error) {
	out, err := r.run("ls-tree", "-r", "-z", rev)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		// "<mode> <type> <sha>\t<path>"
		tab := bytes.IndexByte(rec, '\t')
		if tab < 0 {
			return nil, fmt.Errorf("gitx: unexpected ls-tree record %q", rec)
		}
		f := strings.Fields(string(rec[:tab]))
		if len(f) != 3 {
			return nil, fmt.Errorf("gitx: unexpected ls-tree record %q", rec)
		}
		m[string(rec[tab+1:])] = f[2]
	}
	return m, nil
}

// Show returns the content of path at rev; false if it does not exist there.
func (r Repo) Show(rev, path string) ([]byte, bool, error) {
	if _, err := r.run("cat-file", "-e", rev+":./"+path); err != nil {
		if exitCode(err) != 0 {
			// Absent path (or unknown rev, indistinguishable here): verify rev.
			if _, perr := r.RevParse(rev); perr != nil {
				return nil, false, perr
			}
			return nil, false, nil
		}
		return nil, false, err
	}
	out, err := r.run("cat-file", "blob", rev+":./"+path)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// Tracked reports whether path is tracked in the index.
func (r Repo) Tracked(path string) (bool, error) {
	_, err := r.run("ls-files", "--error-unmatch", "--", path)
	if err == nil {
		return true, nil
	}
	if exitCode(err) == 1 {
		return false, nil
	}
	return false, err
}

// CommitURLTemplate returns a http(s) commit URL template with a {sha}
// placeholder derived from the origin remote, or "".
func (r Repo) CommitURLTemplate() string {
	out, err := r.run("remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return commitURLFromRemote(strings.TrimSpace(string(out)))
}

func commitURLFromRemote(remote string) string {
	scheme, host, path, ok := ParseRemote(remote)
	if !ok {
		return ""
	}
	return scheme + "://" + host + "/" + path + "/commit/{sha}"
}

// ParseRemote splits a git remote URL (https, http, ssh:// or scp-like
// user@host:path forms) into the web scheme, host and "owner/repo" path
// (".git" suffix stripped). ssh remotes map to the "https" scheme and lose
// their port. ok is false when the remote is not of that shape.
func ParseRemote(remote string) (scheme, host, path string, ok bool) {
	scheme = "https"
	switch {
	case strings.HasPrefix(remote, "https://") || strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "ssh://"):
		sep := strings.Index(remote, "://")
		rest := remote[sep+3:]
		slash := strings.IndexByte(rest, '/')
		if slash < 0 {
			return "", "", "", false
		}
		host, path = rest[:slash], rest[slash+1:]
		if at := strings.LastIndexByte(host, '@'); at >= 0 {
			host = host[at+1:]
		}
		if s := remote[:sep]; s == "ssh" {
			// An ssh port is not the web server's: drop it.
			if colon := strings.IndexByte(host, ':'); colon >= 0 {
				host = host[:colon]
			}
		} else {
			scheme = s // http(s) remotes are served by the web server itself
		}
	case strings.Contains(remote, "@") && strings.Contains(remote, ":") && !strings.HasPrefix(remote, "/"):
		// scp-like: user@host:org/repo.git
		rest := remote[strings.IndexByte(remote, '@')+1:]
		colon := strings.IndexByte(rest, ':')
		host, path = rest[:colon], rest[colon+1:]
	default:
		return "", "", "", false
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	if host == "" || path == "" || !strings.Contains(path, "/") {
		return "", "", "", false
	}
	return scheme, host, path, true
}

// Dirty reports whether the working tree under Dir has uncommitted changes,
// including untracked files.
func (r Repo) Dirty() (bool, error) {
	out, err := r.run("status", "--porcelain", "--", ".")
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// Tags lists all tag names, sorted lexically.
func (r Repo) Tags() ([]string, error) {
	out, err := r.run("tag", "--list")
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}

// TagCommit resolves tag to the SHA of the commit it points to, peeling
// annotated tags; false if the tag does not exist.
func (r Repo) TagCommit(tag string) (string, bool, error) {
	out, err := r.run("rev-parse", "--verify", "--quiet", "--end-of-options", "refs/tags/"+tag+"^{commit}")
	if err != nil {
		if exitCode(err) == 1 {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(string(out)), true, nil
}

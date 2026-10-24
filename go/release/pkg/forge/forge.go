// Package forge opens, updates and closes release pull requests and creates
// releases on GitHub and Forgejo through one small interface. Git pushes are
// not handled here.
package forge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PR is a pull request as seen by the release tool.
type PR struct {
	Number                  int
	Head, Base, Title, Body string
	URL                     string
}

// Forge is the operations the release tool needs from a code forge.
type Forge interface {
	// FindOpenPR returns the open PR whose head branch is head, or nil.
	FindOpenPR(head string) (*PR, error)
	CreatePR(head, base, title, body string) (*PR, error)
	UpdatePR(number int, title, body string) error
	// ClosePR comments (when comment is non-empty) and closes the PR.
	ClosePR(number int, comment string) error
	ReleaseExists(tag string) (bool, error)
	// CreateRelease is idempotent: an already existing release is success.
	CreateRelease(tag, name, notes string) error
}

// httpError is a non-success API response. It carries only the method, path,
// status and a short body excerpt, never credentials.
type httpError struct {
	method, path string
	status       int
	body         string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("forge: %s %s: %d %s", e.method, e.path, e.status, e.body)
}

// api is the HTTP plumbing shared by both forges.
type api struct {
	base       string // API root without trailing slash
	owner      string
	repo       string
	authHeader string // full Authorization header value, "" for anonymous
	github     bool
	c          *http.Client
}

func newAPI(base, owner, repo, authHeader string, github bool, c *http.Client) *api {
	if c == nil {
		c = &http.Client{Timeout: 30 * time.Second}
	}
	return &api{base: strings.TrimRight(base, "/"), owner: owner, repo: repo, authHeader: authHeader, github: github, c: c}
}

func (a *api) repoPath(suffix string) string {
	return "/repos/" + url.PathEscape(a.owner) + "/" + url.PathEscape(a.repo) + suffix
}

// do sends a request to path (which may include a query) and decodes a JSON
// response into out when out is non-nil. It returns the response headers.
func (a *api) do(method, path string, in, out any) (http.Header, error) {
	return a.doURL(method, a.base+path, path, in, out)
}

func (a *api) doURL(method, full, path string, in, out any) (http.Header, error) {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("forge: %s %s: %w", method, path, err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, full, body)
	if err != nil {
		return nil, fmt.Errorf("forge: %s %s: %s", method, path, scrub(err))
	}
	req.Header.Set("Accept", "application/json")
	if a.github {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.authHeader != "" {
		req.Header.Set("Authorization", a.authHeader)
	}
	resp, err := a.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forge: %s %s: %s", method, path, scrub(err))
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("forge: %s %s: reading response: %s", method, path, scrub(err))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.Header, &httpError{method: method, path: path, status: resp.StatusCode, body: excerpt(data)}
	}
	if out != nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.Header, fmt.Errorf("forge: %s %s: decoding response: %w", method, path, err)
		}
	}
	return resp.Header, nil
}

// scrub renders a transport error without the request URL, which could carry
// userinfo.
func scrub(err error) string {
	if ue, ok := err.(*url.Error); ok {
		return ue.Err.Error()
	}
	return err.Error()
}

func excerpt(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func statusOf(err error) int {
	if he, ok := err.(*httpError); ok {
		return he.status
	}
	return 0
}

// prJSON is the PR shape common to GitHub and Forgejo responses.
type prJSON struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	Head    struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func (p prJSON) pr() *PR {
	return &PR{Number: p.Number, Head: p.Head.Ref, Base: p.Base.Ref, Title: p.Title, Body: p.Body, URL: p.HTMLURL}
}

// Operations that are identical on both forges, parameterised by the shared api.

func (a *api) createPR(head, base, title, body string) (*PR, error) {
	var out prJSON
	in := map[string]any{"head": head, "base": base, "title": title, "body": body}
	if _, err := a.do("POST", a.repoPath("/pulls"), in, &out); err != nil {
		return nil, err
	}
	pr := out.pr()
	if pr.Head == "" {
		pr.Head = head
	}
	if pr.Base == "" {
		pr.Base = base
	}
	return pr, nil
}

func (a *api) updatePR(number int, title, body string) error {
	_, err := a.do("PATCH", a.repoPath(fmt.Sprintf("/pulls/%d", number)), map[string]any{"title": title, "body": body}, nil)
	return err
}

func (a *api) closePR(number int, comment string) error {
	if comment != "" {
		if _, err := a.do("POST", a.repoPath(fmt.Sprintf("/issues/%d/comments", number)), map[string]any{"body": comment}, nil); err != nil {
			return err
		}
	}
	_, err := a.do("PATCH", a.repoPath(fmt.Sprintf("/pulls/%d", number)), map[string]any{"state": "closed"}, nil)
	return err
}

func (a *api) releaseExists(tag string) (bool, error) {
	_, err := a.do("GET", a.repoPath("/releases/tags/"+url.PathEscape(tag)), nil, nil)
	switch {
	case err == nil:
		return true, nil
	case statusOf(err) == http.StatusNotFound:
		return false, nil
	}
	return false, err
}

func (a *api) createRelease(tag, name, notes string, exists func(*httpError) bool) error {
	in := map[string]any{"tag_name": tag, "name": name, "body": notes}
	_, err := a.do("POST", a.repoPath("/releases"), in, nil)
	if he, ok := err.(*httpError); ok && exists(he) {
		return nil
	}
	return err
}

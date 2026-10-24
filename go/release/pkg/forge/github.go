package forge

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type github struct{ *api }

// NewGitHub returns a GitHub (or GitHub Enterprise) client. apiURL defaults
// to https://api.github.com; c defaults to a client with a 30s timeout.
func NewGitHub(apiURL, owner, repo, token string, c *http.Client) Forge {
	if apiURL == "" {
		apiURL = "https://api.github.com"
	}
	auth := ""
	if token != "" {
		auth = "Bearer " + token
	}
	return &github{newAPI(apiURL, owner, repo, auth, true, c)}
}

var nextLink = regexp.MustCompile(`<([^>]+)>\s*;\s*rel="next"`)

func (g *github) FindOpenPR(head string) (*PR, error) {
	q := url.Values{"state": {"open"}, "head": {g.owner + ":" + head}, "per_page": {"100"}}
	path := g.repoPath("/pulls") + "?" + q.Encode()
	full := g.base + path
	for pages := 0; full != "" && pages < 50; pages++ {
		var prs []prJSON
		hdr, err := g.doURL("GET", full, path, nil, &prs)
		if err != nil {
			return nil, err
		}
		for _, p := range prs {
			if p.Head.Ref == head {
				return p.pr(), nil
			}
		}
		full = ""
		if m := nextLink.FindStringSubmatch(hdr.Get("Link")); m != nil {
			full = m[1]
			if u, err := url.Parse(full); err == nil {
				path = u.RequestURI()
				if strings.HasPrefix(path, "/") && u.Host == "" {
					full = g.base + path
				}
			}
		}
	}
	return nil, nil
}

func (g *github) CreatePR(head, base, title, body string) (*PR, error) {
	return g.createPR(head, base, title, body)
}
func (g *github) UpdatePR(n int, title, body string) error { return g.updatePR(n, title, body) }
func (g *github) ClosePR(n int, comment string) error      { return g.closePR(n, comment) }
func (g *github) ReleaseExists(tag string) (bool, error)   { return g.releaseExists(tag) }

func (g *github) CreateRelease(tag, name, notes string) error {
	return g.createRelease(tag, name, notes, func(e *httpError) bool {
		return e.status == http.StatusUnprocessableEntity && strings.Contains(e.body, "already_exists")
	})
}

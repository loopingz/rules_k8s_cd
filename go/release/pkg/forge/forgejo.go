package forge

import (
	"fmt"
	"net/http"
	"strings"
)

type forgejo struct{ *api }

// NewForgejo returns a Forgejo/Gitea client; the API lives at
// <baseURL>/api/v1. c defaults to a client with a 30s timeout.
func NewForgejo(baseURL, owner, repo, token string, c *http.Client) Forge {
	auth := ""
	if token != "" {
		auth = "token " + token
	}
	return &forgejo{newAPI(strings.TrimRight(baseURL, "/")+"/api/v1", owner, repo, auth, false, c)}
}

func (f *forgejo) FindOpenPR(head string) (*PR, error) {
	for page := 1; page <= 1000; page++ {
		var prs []prJSON
		path := f.repoPath("/pulls") + fmt.Sprintf("?state=open&limit=50&page=%d", page)
		if _, err := f.do("GET", path, nil, &prs); err != nil {
			return nil, err
		}
		if len(prs) == 0 {
			return nil, nil
		}
		for _, p := range prs {
			if p.Head.Ref == head {
				return p.pr(), nil
			}
		}
	}
	return nil, nil
}

func (f *forgejo) CreatePR(head, base, title, body string) (*PR, error) {
	return f.createPR(head, base, title, body)
}
func (f *forgejo) UpdatePR(n int, title, body string) error { return f.updatePR(n, title, body) }
func (f *forgejo) ClosePR(n int, comment string) error      { return f.closePR(n, comment) }
func (f *forgejo) ReleaseExists(tag string) (bool, error)   { return f.releaseExists(tag) }

func (f *forgejo) CreateRelease(tag, name, notes string) error {
	return f.createRelease(tag, name, notes, func(e *httpError) bool { return e.status == http.StatusConflict })
}

package forge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const tok = "s3cr3t-token"

type rec struct {
	method, path, query, auth, accept, ver string
	body                                   map[string]any
}

func server(t *testing.T, h func(r rec, w http.ResponseWriter, req *http.Request)) (*httptest.Server, *[]rec) {
	t.Helper()
	var got []rec
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		r := rec{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("Authorization"),
			req.Header.Get("Accept"), req.Header.Get("X-GitHub-Api-Version"), nil}
		if len(b) > 0 {
			if err := json.Unmarshal(b, &r.body); err != nil {
				t.Errorf("bad json body %q", b)
			}
		}
		got = append(got, r)
		h(r, w, req)
	}))
	t.Cleanup(s.Close)
	return s, &got
}

func pr(n int, head string) string {
	return fmt.Sprintf(`{"number":%d,"title":"T","body":"B","html_url":"http://x/pr/%d","head":{"ref":%q},"base":{"ref":"main"}}`, n, n, head)
}

func TestGitHubOperations(t *testing.T) {
	s, got := server(t, func(r rec, w http.ResponseWriter, req *http.Request) {
		switch {
		case r.method == "GET" && strings.HasSuffix(r.path, "/pulls"):
			fmt.Fprintf(w, "[%s]", pr(7, "release/app"))
		case r.method == "POST" && strings.HasSuffix(r.path, "/pulls"):
			w.WriteHeader(201)
			fmt.Fprint(w, pr(8, "release/app"))
		case r.method == "GET" && strings.Contains(r.path, "/releases/tags/"):
			w.WriteHeader(404)
		default:
			fmt.Fprint(w, "{}")
		}
	})
	g := NewGitHub(s.URL, "o", "r", tok, nil)

	p, err := g.FindOpenPR("release/app")
	if err != nil || p == nil || p.Number != 7 || p.Head != "release/app" || p.Base != "main" || p.URL != "http://x/pr/7" {
		t.Fatalf("FindOpenPR = %+v, %v", p, err)
	}
	r0 := (*got)[0]
	if r0.path != "/repos/o/r/pulls" || r0.query != "head=o%3Arelease%2Fapp&per_page=100&state=open" ||
		r0.auth != "Bearer "+tok || r0.accept != "application/vnd.github+json" || r0.ver != "2022-11-28" {
		t.Errorf("find request = %+v", r0)
	}

	p, err = g.CreatePR("release/app", "main", "title", "body")
	if err != nil || p.Number != 8 {
		t.Fatalf("CreatePR = %+v, %v", p, err)
	}
	b := (*got)[1].body
	if (*got)[1].method != "POST" || b["head"] != "release/app" || b["base"] != "main" || b["title"] != "title" || b["body"] != "body" {
		t.Errorf("create request = %+v", (*got)[1])
	}

	if err := g.UpdatePR(8, "t2", "b2"); err != nil {
		t.Fatal(err)
	}
	if r := (*got)[2]; r.method != "PATCH" || r.path != "/repos/o/r/pulls/8" || r.body["title"] != "t2" || r.body["body"] != "b2" {
		t.Errorf("update request = %+v", r)
	}

	if err := g.ClosePR(8, "superseded"); err != nil {
		t.Fatal(err)
	}
	if r := (*got)[3]; r.method != "POST" || r.path != "/repos/o/r/issues/8/comments" || r.body["body"] != "superseded" {
		t.Errorf("comment request = %+v", r)
	}
	if r := (*got)[4]; r.method != "PATCH" || r.path != "/repos/o/r/pulls/8" || r.body["state"] != "closed" {
		t.Errorf("close request = %+v", r)
	}

	ok, err := g.ReleaseExists("app@v1.0.0")
	if err != nil || ok {
		t.Fatalf("ReleaseExists 404 = %v, %v", ok, err)
	}
	if err := g.CreateRelease("app@v1.0.0", "app 1.0.0", "notes"); err != nil {
		t.Fatal(err)
	}
	r := (*got)[len(*got)-1]
	if r.method != "POST" || r.path != "/repos/o/r/releases" || r.body["tag_name"] != "app@v1.0.0" || r.body["name"] != "app 1.0.0" || r.body["body"] != "notes" {
		t.Errorf("release request = %+v", r)
	}
}

func TestGitHubFindOpenPRNoneAndPagination(t *testing.T) {
	var srv *httptest.Server
	srv, got := server(t, func(r rec, w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("page") == "2" {
			fmt.Fprintf(w, "[%s]", pr(9, "release/app"))
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/repos/o/r/pulls?state=open&page=2>; rel="next"`, srv.URL))
		fmt.Fprintf(w, "[%s]", pr(1, "other"))
	})
	g := NewGitHub(srv.URL, "o", "r", tok, nil)
	p, err := g.FindOpenPR("release/app")
	if err != nil || p == nil || p.Number != 9 {
		t.Fatalf("got %+v, %v", p, err)
	}
	if len(*got) != 2 {
		t.Errorf("requests = %d", len(*got))
	}
	p, err = g.FindOpenPR("nope")
	if err != nil || p != nil {
		t.Errorf("want nil, got %+v %v", p, err)
	}
}

func TestGitHubCreateReleaseIdempotent(t *testing.T) {
	s, _ := server(t, func(r rec, w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(422)
		fmt.Fprint(w, `{"message":"Validation Failed","errors":[{"resource":"Release","code":"already_exists","field":"tag_name"}]}`)
	})
	if err := NewGitHub(s.URL, "o", "r", tok, nil).CreateRelease("t", "n", "b"); err != nil {
		t.Errorf("already_exists should be success: %v", err)
	}
	s2, _ := server(t, func(r rec, w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(422)
		fmt.Fprint(w, `{"message":"other"}`)
	})
	if err := NewGitHub(s2.URL, "o", "r", tok, nil).CreateRelease("t", "n", "b"); err == nil {
		t.Error("other 422 should fail")
	}
}

func TestForgejoOperations(t *testing.T) {
	s, got := server(t, func(r rec, w http.ResponseWriter, req *http.Request) {
		switch {
		case r.method == "GET" && strings.HasSuffix(r.path, "/pulls"):
			switch req.URL.Query().Get("page") {
			case "1":
				fmt.Fprintf(w, "[%s]", pr(1, "feature/x"))
			case "2":
				fmt.Fprintf(w, "[%s]", pr(2, "release/app"))
			default:
				fmt.Fprint(w, "[]")
			}
		case r.method == "POST" && strings.HasSuffix(r.path, "/pulls"):
			w.WriteHeader(201)
			fmt.Fprint(w, pr(3, "release/app"))
		case r.method == "GET" && strings.Contains(r.path, "/releases/tags/"):
			fmt.Fprint(w, "{}")
		case r.method == "POST" && strings.HasSuffix(r.path, "/releases"):
			w.WriteHeader(409)
		default:
			fmt.Fprint(w, "{}")
		}
	})
	f := NewForgejo(s.URL, "o", "r", tok, nil)

	p, err := f.FindOpenPR("release/app")
	if err != nil || p == nil || p.Number != 2 {
		t.Fatalf("FindOpenPR = %+v, %v", p, err)
	}
	if len(*got) != 2 || (*got)[1].path != "/api/v1/repos/o/r/pulls" || (*got)[1].query != "state=open&limit=50&page=2" || (*got)[1].auth != "token "+tok {
		t.Errorf("requests = %+v", *got)
	}
	p, err = f.FindOpenPR("missing")
	if err != nil || p != nil {
		t.Errorf("missing = %+v, %v", p, err)
	}

	*got = nil
	if _, err := f.CreatePR("release/app", "main", "t", "b"); err != nil {
		t.Fatal(err)
	}
	if err := f.UpdatePR(3, "t2", "b2"); err != nil {
		t.Fatal(err)
	}
	if err := f.ClosePR(3, "bye"); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /api/v1/repos/o/r/pulls", "PATCH /api/v1/repos/o/r/pulls/3",
		"POST /api/v1/repos/o/r/issues/3/comments", "PATCH /api/v1/repos/o/r/pulls/3"}
	for i, w := range want {
		if g := (*got)[i].method + " " + (*got)[i].path; g != w {
			t.Errorf("req %d = %s, want %s", i, g, w)
		}
	}
	if (*got)[3].body["state"] != "closed" || (*got)[2].body["body"] != "bye" {
		t.Errorf("bodies: %+v %+v", (*got)[2], (*got)[3])
	}
	if (*got)[0].accept == "application/vnd.github+json" {
		t.Error("forgejo must not send github accept")
	}

	ok, err := f.ReleaseExists("t")
	if err != nil || !ok {
		t.Errorf("ReleaseExists = %v, %v", ok, err)
	}
	if err := f.CreateRelease("t", "n", "b"); err != nil {
		t.Errorf("409 should be success: %v", err)
	}
}

func TestReleaseExistsOtherStatusIsError(t *testing.T) {
	s, _ := server(t, func(r rec, w http.ResponseWriter, req *http.Request) { w.WriteHeader(500) })
	if _, err := NewForgejo(s.URL, "o", "r", tok, nil).ReleaseExists("t"); err == nil {
		t.Error("500 must be an error")
	}
}

func TestErrorsNeverContainToken(t *testing.T) {
	s, _ := server(t, func(r rec, w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, strings.Repeat("x", 500))
	})
	for _, f := range []Forge{NewGitHub(s.URL, "o", "r", tok, nil), NewForgejo(s.URL, "o", "r", tok, nil)} {
		_, e1 := f.FindOpenPR("h")
		_, e2 := f.CreatePR("h", "b", "t", "b")
		e3 := f.UpdatePR(1, "t", "b")
		e4 := f.ClosePR(1, "c")
		_, e5 := f.ReleaseExists("t")
		e6 := f.CreateRelease("t", "n", "b")
		for _, e := range []error{e1, e2, e3, e4, e5, e6} {
			if e == nil {
				t.Fatal("want error")
			}
			if strings.Contains(e.Error(), tok) {
				t.Errorf("token leaked: %v", e)
			}
			if !strings.Contains(e.Error(), "403") || !strings.HasPrefix(e.Error(), "forge: ") {
				t.Errorf("bad error format: %v", e)
			}
			if len(e.Error()) > 300 {
				t.Errorf("body not truncated: %d", len(e.Error()))
			}
		}
	}
	// Transport error with userinfo in the URL must not leak either.
	bad := NewGitHub("http://user:"+tok+"@127.0.0.1:1", "o", "r", tok, nil)
	if _, err := bad.FindOpenPR("h"); err == nil || strings.Contains(err.Error(), tok) {
		t.Errorf("transport error: %v", err)
	}
}

func TestDetect(t *testing.T) {
	envOf := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	tests := []struct {
		name   string
		env    map[string]string
		remote string
		flags  Config
		want   Config
		err    string
	}{
		{name: "github remote", remote: "git@github.com:acme/widgets.git",
			want: Config{Kind: "github", URL: "https://api.github.com", Owner: "acme", Repo: "widgets"}},
		{name: "github actions", env: map[string]string{"GITHUB_ACTIONS": "true", "GITHUB_API_URL": "https://ghe.example/api/v3", "GITHUB_TOKEN": "g"},
			remote: "https://ghe.example/acme/widgets",
			want:   Config{Kind: "github", URL: "https://ghe.example/api/v3", Owner: "acme", Repo: "widgets", Token: "g"}},
		{name: "github actions default api", env: map[string]string{"GITHUB_ACTIONS": "true"}, remote: "https://x/a/b",
			want: Config{Kind: "github", URL: "https://api.github.com", Owner: "a", Repo: "b"}},
		{name: "forgejo actions", env: map[string]string{"FORGEJO_ACTIONS": "true", "GITHUB_ACTIONS": "true", "GITHUB_SERVER_URL": "https://git.example.com/", "FORGEJO_TOKEN": "f"},
			remote: "https://git.example.com/acme/widgets.git",
			want:   Config{Kind: "forgejo", URL: "https://git.example.com", Owner: "acme", Repo: "widgets", Token: "f"}},
		{name: "gitea actions flag url", env: map[string]string{"GITEA_ACTIONS": "true", "GITHUB_SERVER_URL": "https://ignored"},
			remote: "ssh://git@git.example.com:2222/acme/widgets.git", flags: Config{URL: "https://forge.example"},
			want: Config{Kind: "forgejo", URL: "https://forge.example", Owner: "acme", Repo: "widgets"}},
		{name: "flag kind forgejo uses remote host", remote: "git@git.example.com:acme/widgets.git", flags: Config{Kind: "forgejo"},
			want: Config{Kind: "forgejo", URL: "https://git.example.com", Owner: "acme", Repo: "widgets"}},
		{name: "enterprise flag url", remote: "git@ghe.example:acme/widgets.git", flags: Config{Kind: "github", URL: "https://ghe.example"},
			want: Config{Kind: "github", URL: "https://ghe.example/api/v3", Owner: "acme", Repo: "widgets"}},
		{name: "repo flag and token precedence", env: map[string]string{"FORGE_TOKEN": "a", "GITHUB_TOKEN": "b"},
			flags: Config{Kind: "github", Repo: "o/n"},
			want:  Config{Kind: "github", URL: "https://api.github.com", Owner: "o", Repo: "n", Token: "a"}},
		{name: "flag token wins", env: map[string]string{"FORGE_TOKEN": "a"}, remote: "https://github.com/o/n", flags: Config{Token: "z"},
			want: Config{Kind: "github", URL: "https://api.github.com", Owner: "o", Repo: "n", Token: "z"}},
		{name: "unknown host", remote: "https://git.example.com/o/n", err: "--forge"},
		{name: "bad kind", remote: "https://github.com/o/n", flags: Config{Kind: "gitlab"}, err: "gitlab"},
		{name: "no repo", flags: Config{Kind: "github"}, err: "--repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Detect(envOf(tt.env), tt.remote, tt.flags)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("err = %v, want containing %q", err, tt.err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("got %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

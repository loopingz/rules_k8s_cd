package forge

import (
	"errors"
	"fmt"
	"strings"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
)

// Config selects and addresses a forge. For Kind "github", URL is the API
// root; for "forgejo" it is the web base URL.
//
// As flags to Detect: Kind ("github"/"forgejo") overrides detection; URL is
// the forge web URL (GitHub Enterprise: API at <url>/api/v3); Repo may be
// "owner/name" instead of setting Owner and Repo; Token overrides the env.
type Config struct {
	Kind, URL, Owner, Repo, Token string
}

// Detect resolves the forge configuration from flags, the environment and the
// git remote URL.
func Detect(env func(string) string, remoteURL string, flags Config) (Config, error) {
	cfg := flags
	scheme, host, path, remoteOK := gitx.ParseRemote(remoteURL)

	if cfg.Owner == "" && strings.Contains(cfg.Repo, "/") {
		i := strings.LastIndex(cfg.Repo, "/")
		cfg.Owner, cfg.Repo = cfg.Repo[:i], cfg.Repo[i+1:]
	}
	if cfg.Owner == "" || cfg.Repo == "" {
		if !remoteOK {
			return Config{}, errors.New("forge: cannot determine owner/repo: pass --repo owner/name")
		}
		i := strings.LastIndex(path, "/")
		if cfg.Owner == "" {
			cfg.Owner = path[:i]
		}
		if cfg.Repo == "" {
			cfg.Repo = path[i+1:]
		}
	}

	flagURL := strings.TrimRight(flags.URL, "/")
	switch strings.ToLower(cfg.Kind) {
	case "github", "forgejo":
		cfg.Kind = strings.ToLower(cfg.Kind)
	case "gitea":
		cfg.Kind = "forgejo"
	case "":
		switch {
		case on(env("FORGEJO_ACTIONS")) || on(env("GITEA_ACTIONS")):
			cfg.Kind = "forgejo"
		case env("GITHUB_ACTIONS") == "true":
			cfg.Kind = "github"
		case remoteOK && host == "github.com":
			cfg.Kind = "github"
		default:
			return Config{}, errors.New("forge: cannot detect the forge: pass --forge github|forgejo")
		}
	default:
		return Config{}, fmt.Errorf("forge: unknown forge kind %q (want github or forgejo)", cfg.Kind)
	}

	switch cfg.Kind {
	case "github":
		switch {
		case flagURL != "":
			cfg.URL = githubAPIFromWeb(flagURL)
		case env("GITHUB_ACTIONS") == "true":
			cfg.URL = "https://api.github.com"
			if u := env("GITHUB_API_URL"); u != "" {
				cfg.URL = strings.TrimRight(u, "/")
			}
		case remoteOK && host != "github.com":
			cfg.URL = githubAPIFromWeb(scheme + "://" + host)
		default:
			cfg.URL = "https://api.github.com"
		}
	case "forgejo":
		switch {
		case flagURL != "":
			cfg.URL = flagURL
		case env("GITHUB_SERVER_URL") != "":
			cfg.URL = strings.TrimRight(env("GITHUB_SERVER_URL"), "/")
		case remoteOK:
			cfg.URL = scheme + "://" + host
		default:
			return Config{}, errors.New("forge: cannot determine the Forgejo URL: pass --forge-url")
		}
	}

	if cfg.Token == "" {
		for _, k := range []string{"FORGE_TOKEN", "GITHUB_TOKEN", "FORGEJO_TOKEN"} {
			if v := env(k); v != "" {
				cfg.Token = v
				break
			}
		}
	}
	return cfg, nil
}

func on(v string) bool {
	v = strings.ToLower(v)
	return v != "" && v != "false" && v != "0"
}

// githubAPIFromWeb maps a GitHub web URL to its API root: github.com to
// api.github.com, Enterprise hosts to <url>/api/v3.
func githubAPIFromWeb(u string) string {
	u = strings.TrimRight(u, "/")
	switch {
	case u == "https://github.com" || u == "http://github.com":
		return "https://api.github.com"
	case strings.HasSuffix(u, "/api/v3") || strings.HasPrefix(u, "https://api.github.com"):
		return u
	}
	return u + "/api/v3"
}

package conventional

import (
	"strings"
	"testing"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

func v(s string) semver.Version {
	x, err := semver.Parse(s)
	if err != nil {
		panic(err)
	}
	return x
}

func TestParse(t *testing.T) {
	ra := v("3.0.0")
	tests := []struct {
		name string
		c    gitx.Commit
		want Parsed
	}{
		{"feat", gitx.Commit{Subject: "feat: add x"}, Parsed{Type: "feat", Description: "add x", Conventional: true}},
		{"fix scope", gitx.Commit{Subject: "fix(api): bug"}, Parsed{Type: "fix", Scope: "api", Description: "bug", Conventional: true}},
		{"bang", gitx.Commit{Subject: "feat(a)!: drop"}, Parsed{Type: "feat", Scope: "a", Description: "drop", Breaking: true, Conventional: true}},
		{"bang no scope", gitx.Commit{Subject: "fix!: drop"}, Parsed{Type: "fix", Description: "drop", Breaking: true, Conventional: true}},
		{"footer", gitx.Commit{Subject: "fix: y", Body: "text\n\nBREAKING CHANGE: gone"}, Parsed{Type: "fix", Description: "y", Breaking: true, Conventional: true}},
		{"footer dash", gitx.Commit{Subject: "fix: y", Body: "BREAKING-CHANGE: gone"}, Parsed{Type: "fix", Description: "y", Breaking: true, Conventional: true}},
		{"footer wrong case", gitx.Commit{Subject: "fix: y", Body: "breaking change: gone"}, Parsed{Type: "fix", Description: "y", Conventional: true}},
		{"release-as", gitx.Commit{Subject: "chore: r", Body: "release-AS: 3.0.0"}, Parsed{Type: "chore", Description: "r", ReleaseAs: &ra, Conventional: true}},
		{"plain", gitx.Commit{Subject: "Fix things"}, Parsed{Description: "Fix things"}},
		{"merge", gitx.Commit{Subject: "Merge branch 'x' into main"}, Parsed{Description: "Merge branch 'x' into main"}},
		{"git revert", gitx.Commit{Subject: `Revert "feat: x"`}, Parsed{Description: `Revert "feat: x"`}},
		{"revert type", gitx.Commit{Subject: "revert: x"}, Parsed{Type: "revert", Description: "x", Conventional: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.c)
			if got.Type != tt.want.Type || got.Scope != tt.want.Scope || got.Description != tt.want.Description ||
				got.Breaking != tt.want.Breaking || got.Conventional != tt.want.Conventional {
				t.Fatalf("got %+v want %+v", got, tt.want)
			}
			if (got.ReleaseAs == nil) != (tt.want.ReleaseAs == nil) || (got.ReleaseAs != nil && *got.ReleaseAs != *tt.want.ReleaseAs) {
				t.Fatalf("ReleaseAs got %v want %v", got.ReleaseAs, tt.want.ReleaseAs)
			}
		})
	}
}

func TestParseInvalidReleaseAs(t *testing.T) {
	p := Parse(gitx.Commit{Subject: "fix: a", Body: "Release-As: banana"})
	if p.ReleaseAs != nil || len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0], "banana") {
		t.Fatalf("got %+v", p)
	}
}

func TestBumpFor(t *testing.T) {
	big, small := v("1.2.3"), v("0.2.3")
	tests := []struct {
		subj, body string
		big, small semver.Bump
	}{
		{"feat!: x", "", semver.Major, semver.Minor},
		{"fix: x", "BREAKING CHANGE: y", semver.Major, semver.Minor},
		{"feat: x", "", semver.Minor, semver.Patch},
		{"fix: x", "", semver.Patch, semver.Patch},
		{"perf: x", "", semver.Patch, semver.Patch},
		{"revert: x", "", semver.Patch, semver.Patch},
		{"Merge branch x", "", semver.Patch, semver.Patch},
		{"chore: x", "", semver.None, semver.None},
		{"docs: x", "", semver.None, semver.None},
		{"style: x", "", semver.None, semver.None},
		{"test: x", "", semver.None, semver.None},
		{"ci: x", "", semver.None, semver.None},
		{"build: x", "", semver.None, semver.None},
		{"refactor: x", "", semver.None, semver.None},
		{"chore!: x", "", semver.Major, semver.Minor},
		{"wip: x", "", semver.Patch, semver.Patch},
		{"FEAT: x", "", semver.Minor, semver.Patch},
	}
	for _, tt := range tests {
		p := Parse(gitx.Commit{Subject: tt.subj, Body: tt.body})
		if got := BumpFor(p, big); got != tt.big {
			t.Errorf("%q >=1: got %v want %v", tt.subj, got, tt.big)
		}
		if got := BumpFor(p, small); got != tt.small {
			t.Errorf("%q <1: got %v want %v", tt.subj, got, tt.small)
		}
	}
}

func ps(subjects ...string) []Parsed {
	var out []Parsed
	for _, s := range subjects {
		out = append(out, Parse(gitx.Commit{Subject: s}))
	}
	return out
}

func TestNext(t *testing.T) {
	cur := v("1.2.3")
	initial := v("0.1.0")
	ra := func(s string) Parsed {
		return Parse(gitx.Commit{Subject: "chore: r", Body: "Release-As: " + s})
	}
	tests := []struct {
		name     string
		current  *semver.Version
		commits  []Parsed
		want     string
		release  bool
		warnPart string
	}{
		{"nil current releasable", nil, ps("chore: a", "feat: b"), "0.1.0", true, ""},
		{"nil current none", nil, ps("chore: a"), "", false, ""},
		{"nil empty", nil, nil, "", false, ""},
		{"nil release-as", nil, []Parsed{ra("2.0.0")}, "2.0.0", true, ""},
		{"patch", &cur, ps("fix: a"), "1.2.4", true, ""},
		{"minor wins", &cur, ps("fix: a", "feat: b"), "1.3.0", true, ""},
		{"major wins", &cur, ps("feat!: a", "feat: b"), "2.0.0", true, ""},
		{"none", &cur, ps("docs: a", "chore: b"), "", false, ""},
		{"release-as wins", &cur, append(ps("feat!: a"), ra("1.9.0")), "1.9.0", true, ""},
		{"release-as greatest", &cur, []Parsed{ra("1.9.0"), ra("1.5.0")}, "1.9.0", true, ""},
		{"release-as not greater", &cur, append(ps("fix: a"), ra("1.2.3")), "1.2.4", true, "not greater"},
		{"release-as lower only", &cur, []Parsed{ra("1.0.0")}, "", false, "not greater"},
		{"release-as invalid", &cur, []Parsed{ra("x")}, "", false, "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rel, warns := Next(tt.current, initial, tt.commits)
			if rel != tt.release {
				t.Fatalf("release=%v want %v (%v)", rel, tt.release, warns)
			}
			if rel && got.String() != tt.want {
				t.Fatalf("got %s want %s", got, tt.want)
			}
			if tt.warnPart == "" && len(warns) != 0 {
				t.Fatalf("unexpected warnings %v", warns)
			}
			if tt.warnPart != "" && !strings.Contains(strings.Join(warns, "\n"), tt.warnPart) {
				t.Fatalf("warnings %v lack %q", warns, tt.warnPart)
			}
		})
	}
}

func TestNextPreOne(t *testing.T) {
	cur := v("0.4.0")
	got, rel, _ := Next(&cur, v("0.1.0"), ps("feat!: a"))
	if !rel || got.String() != "0.5.0" {
		t.Fatalf("got %s %v", got, rel)
	}
}

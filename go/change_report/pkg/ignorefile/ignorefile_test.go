package ignorefile

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	a = "0123456789abcdef0123456789abcdef01234567"
	b = "89abcdef0123456789abcdef0123456789abcdef"
)

func TestParse(t *testing.T) {
	f, warns := Parse(".release-ignore", []byte(`# reformat
`+a+`
`+strings.ToUpper(b)+` api cli //tools:gen  # a pin
docs/**
0123abc api
two words here
deadbeef
`))
	if !reflect.DeepEqual(f.Commits, map[string][]string{a: nil, b: {"api", "cli", "//tools:gen"}}) {
		t.Errorf("commits %v", f.Commits)
	}
	if !reflect.DeepEqual(f.Paths, []string{"docs/**"}) {
		t.Errorf("paths %v", f.Paths)
	}
	if len(warns) != 3 || !strings.Contains(warns[0], ":5: commit 0123abc is not a full SHA") || !strings.Contains(warns[1], ":6:") ||
		!strings.Contains(warns[2], ":7: commit deadbeef") {
		t.Errorf("warnings %q", warns)
	}
}

func TestIgnores(t *testing.T) {
	f, _ := Parse("x", []byte(a+"\n"+b+" api\n"))
	for _, tc := range []struct {
		sha, body string
		names     []string
		want      bool
	}{
		{a, "", []string{"cli"}, true},
		{b, "", []string{"api"}, true},
		{b, "", []string{"cli"}, false},
		{"c", "fix: x\n\nRelease-Ignore: cli, web", []string{"cli"}, true},
		{"c", "fix: x\n\nRelease-Ignore: cli, web", []string{"api"}, false},
		{"c", "fix: x\n\nrelease-ignore: *", []string{"api"}, true},
		{"c", "fix: x\n\nRelease-Ignore:", []string{"api"}, true},
		{"c", "fix: x\n\nThe Release-Ignore: trailer", []string{"api"}, false},
	} {
		if got := f.Ignores(tc.sha, tc.body, tc.names...); got != tc.want {
			t.Errorf("%s %q %v: %v", tc.sha, tc.body, tc.names, got)
		}
	}
}

func TestCheck(t *testing.T) {
	f, _ := Parse("x", []byte(a+" api ghost //a:a //b:b\n"+b+"\n"))
	warns := f.Check(func(sha string) bool { return sha == a }, false, func(n string) bool { return n == "api" })
	sort.Strings(warns)
	if len(warns) != 2 || !strings.Contains(warns[0], "commit "+a+": no release component ghost") || !strings.Contains(warns[1], b+" is not in the repository") {
		t.Errorf("components: %q", warns)
	}
	warns = f.Check(nil, true, func(n string) bool { return n == "//a:a" })
	if len(warns) != 1 || !strings.Contains(warns[0], "no target //b:b") {
		t.Errorf("labels: %q", warns)
	}
}

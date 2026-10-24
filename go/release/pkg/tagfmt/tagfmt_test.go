package tagfmt

import (
	"testing"

	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

func TestParserAndFormat(t *testing.T) {
	tests := []struct {
		format, name, tag string
		want              string // "" = not owned
	}{
		{"{name}@v{version}", "api", "api@v1.2.3", "1.2.3"},
		{"{name}@v{version}", "api", "web@v1.2.3", ""},
		{"{name}@v{version}", "api", "api@v1.2", ""},
		{"v{version}", "api", "v4.1.0", "4.1.0"},
		{"v{version}", "api", "4.1.0", ""},
		{"{name}-{version}", "api", "api-gateway-1.0.0", ""},
		{"{name}-{version}", "api", "api-1.0.0", "1.0.0"},
		{"v{version}-rc", "x", "v1.0.0-rc", "1.0.0"},
		{"v{version}-rc", "x", "v1.0.0", ""},
		{"v{version}", "x", "v", ""},
	}
	for _, tc := range tests {
		v, ok := Parser(tc.format, tc.name)(tc.tag)
		if ok != (tc.want != "") || (ok && v.String() != tc.want) {
			t.Errorf("Parser(%q,%q)(%q) = %v,%v want %q", tc.format, tc.name, tc.tag, v, ok, tc.want)
		}
		if ok {
			if got := Format(tc.format, tc.name, v); got != tc.tag {
				t.Errorf("Format round trip = %q want %q", got, tc.tag)
			}
		}
	}
	if got := Format("{name}@v{version}", "api", semver.Version{Major: 1, Minor: 2, Patch: 3}); got != "api@v1.2.3" {
		t.Errorf("Format = %q", got)
	}
}

func TestValidate(t *testing.T) {
	for _, f := range []string{"{name}@v{version}", "v{version}"} {
		if err := Validate(f, "api"); err != nil {
			t.Errorf("Validate(%q) = %v", f, err)
		}
	}
	for _, f := range []string{"{name}", "{version}", "{name}{version}x"} {
		name := "api"
		if f == "{name}{version}x" {
			name = ""
		}
		if err := Validate(f, name); err == nil {
			t.Errorf("Validate(%q,%q) want error", f, name)
		}
	}
}

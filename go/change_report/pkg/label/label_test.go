package label

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		in       string
		repo     string
		pkg      string
		name     string
		path     string
		str      string
		wantErr  bool
		skipPath bool
	}{
		{in: "@@//pkg:manifests/a.yaml", repo: "", pkg: "pkg", name: "manifests/a.yaml", path: "pkg/manifests/a.yaml", str: "//pkg:manifests/a.yaml"},
		{in: "//:BUILD", repo: "", pkg: "", name: "BUILD", path: "BUILD", str: "//:BUILD"},
		{in: "//a/b", repo: "", pkg: "a/b", name: "b", path: "a/b/b", str: "//a/b:b"},
		{in: "@@rules_python++config+pypi__build//:pkg", repo: "rules_python++config+pypi__build", pkg: "", name: "pkg", str: "@@rules_python++config+pypi__build//:pkg", skipPath: true},
		{in: "@@//pkg:x", repo: "", pkg: "pkg", name: "x", path: "pkg/x", str: "//pkg:x"},
		{in: "@repo//x:y", repo: "repo", pkg: "x", name: "y", str: "@@repo//x:y", skipPath: true},
		{in: "@@repo+//x:y", repo: "repo+", pkg: "x", name: "y", str: "@@repo+//x:y", skipPath: true},
		{in: "garbage", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := Parse(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %+v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tc.in, err)
			}
			if got.Repo != tc.repo || got.Pkg != tc.pkg || got.Name != tc.name {
				t.Errorf("Parse(%q) = {%q %q %q}, want {%q %q %q}", tc.in, got.Repo, got.Pkg, got.Name, tc.repo, tc.pkg, tc.name)
			}
			if got.String() != tc.str {
				t.Errorf("String() = %q, want %q", got.String(), tc.str)
			}
			if !tc.skipPath && got.Path() != tc.path {
				t.Errorf("Path() = %q, want %q", got.Path(), tc.path)
			}
		})
	}
}

func TestIsMain(t *testing.T) {
	main, _ := Parse("//a:b")
	if !main.IsMain() {
		t.Errorf("IsMain() = false for main-repo label")
	}
	ext, _ := Parse("@repo//a:b")
	if ext.IsMain() {
		t.Errorf("IsMain() = true for external label")
	}
}

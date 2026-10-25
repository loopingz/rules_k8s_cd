package stmt

import (
	"reflect"
	"testing"
)

const module = `module(name = "x")

# a comment
bazel_dep(name = "rules_go", version = "0.50.0")
oci = use_extension("@rules_oci//oci:extensions.bzl", "oci")
oci.pull(
    name = "nginx",  # trailing comment
    digest = "sha256:1",
    platforms = [
        "linux/amd64",
    ],
)
use_repo(oci, "nginx")
http_file = use_repo_rule("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")
http_file(name = "kubectl", url = "https://x/(y)")
VERSION = """a
(b"""
`

func TestSplit(t *testing.T) {
	got := Split(module)
	var calls, assigns []string
	for _, s := range got {
		calls = append(calls, s.Call)
		assigns = append(assigns, s.Assign)
	}
	if want := []string{"module", "bazel_dep", "use_extension", "oci.pull", "use_repo", "use_repo_rule", "http_file", ""}; !reflect.DeepEqual(calls, want) {
		t.Errorf("calls %q", calls)
	}
	if want := []string{"", "", "oci", "", "", "http_file", "", "VERSION"}; !reflect.DeepEqual(assigns, want) {
		t.Errorf("assigns %q", assigns)
	}
	if got[3].Text != `oci.pull(name = "nginx", digest = "sha256:1", platforms = ["linux/amd64"])` {
		t.Errorf("text %q", got[3].Text)
	}
	if got[3].Kwarg("name") != "nginx" || got[6].Kwarg("name") != "kubectl" {
		t.Errorf("kwargs %q %q", got[3].Kwarg("name"), got[6].Kwarg("name"))
	}
	if got[2].Positional(0) != "@rules_oci//oci:extensions.bzl" || got[2].Positional(1) != "oci" {
		t.Errorf("positional %q %q", got[2].Positional(0), got[2].Positional(1))
	}
}

func TestChangedIgnoresFormatting(t *testing.T) {
	after := `module(name = "x")
bazel_dep(name = "rules_go", version = "0.50.0")  # bumped soon
oci = use_extension("@rules_oci//oci:extensions.bzl", "oci")
oci.pull(name = "nginx", digest = "sha256:2", platforms = ["linux/amd64"])
use_repo(oci, "nginx")
http_file = use_repo_rule("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")
http_file(
    name = "kubectl",
    url = "https://x/(y)",
)
VERSION = """a
(b"""
`
	ch := Changed(Split(module), Split(after))
	if len(ch) != 2 || ch[0].Call != "oci.pull" || ch[1].Call != "oci.pull" {
		t.Errorf("changed %+v", ch)
	}
}

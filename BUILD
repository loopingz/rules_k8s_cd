load("@gazelle//:def.bzl", "gazelle")
load("//lib:release.bzl", "release_component")

gazelle(name = "gazelle")

gazelle(
    name = "gazelle-update-repos",
    args = [
        "-from_file=ci/go/go.mod",
        "-to_macro=deps.bzl%go_dependencies",
        "-prune",
        "-build_file_proto_mode=disable_global",
    ],
    command = "update-repos",
)

release_component(
    name = "rules_k8s_cd",
    bump_files = {
        # Group 1 is the version in `module(..., version = "X.Y.Z")`; the
        # bazel_dep versions after the closing parenthesis are not matched.
        "MODULE.bazel": r'(?s)module\([^)]*?\bversion = "([^"]+)"',
    },
    changelog = "CHANGELOG.md",
    release_targets = ["//tools/release:release_notes_snippet"],
    tag_format = "v{version}",
    targets_query = "//...",
    version_file = "VERSION",
)

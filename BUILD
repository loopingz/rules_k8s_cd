#load("@com_github_bazelbuild_buildtools//buildifier:def.bzl", "buildifier")
load("@gazelle//:def.bzl", "gazelle")

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

# buildifier(
#     name = "buildifier",
# )

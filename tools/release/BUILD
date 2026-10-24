load("@rules_shell//shell:sh_binary.bzl", "sh_binary")
load("@rules_shell//shell:sh_test.bzl", "sh_test")

sh_binary(
    name = "release_notes_snippet",
    srcs = ["release_notes_snippet.sh"],
    visibility = ["//visibility:public"],
)

sh_test(
    name = "release_notes_snippet_test",
    size = "small",
    srcs = ["release_notes_snippet_test.sh"],
    args = ["$(location release_notes_snippet.sh)"],
    data = ["release_notes_snippet.sh"],
)

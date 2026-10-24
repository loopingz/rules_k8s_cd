load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "tagfmt",
    srcs = ["tagfmt.go"],
    importpath = "loopingz.com/rules_k8s_cd/release/pkg/tagfmt",
    visibility = ["//go/release:__subpackages__"],
    deps = ["//go/release/pkg/semver"],
)

go_test(
    name = "tagfmt_test",
    srcs = ["tagfmt_test.go"],
    embed = [":tagfmt"],
    deps = ["//go/release/pkg/semver"],
)

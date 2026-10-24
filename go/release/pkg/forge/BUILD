load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "forge",
    srcs = [
        "detect.go",
        "forge.go",
        "forgejo.go",
        "github.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/release/pkg/forge",
    visibility = ["//go/release:__subpackages__"],
    deps = ["//go/change_report/pkg/gitx"],
)

go_test(
    name = "forge_test",
    srcs = ["forge_test.go"],
    embed = [":forge"],
)

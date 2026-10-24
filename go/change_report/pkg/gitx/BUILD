load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "gitx",
    srcs = [
        "git.go",
        "log.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/gitx",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
)

go_test(
    name = "gitx_test",
    srcs = ["git_test.go"],
    embed = [":gitx"],
)

load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "cliutil",
    srcs = ["cliutil.go"],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/cliutil",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
)

go_test(
    name = "cliutil_test",
    srcs = ["cliutil_test.go"],
    embed = [":cliutil"],
)

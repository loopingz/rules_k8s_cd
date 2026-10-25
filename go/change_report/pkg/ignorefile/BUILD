load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "ignorefile",
    srcs = ["ignorefile.go"],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/ignorefile",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
)

go_test(
    name = "ignorefile_test",
    srcs = ["ignorefile_test.go"],
    embed = [":ignorefile"],
)

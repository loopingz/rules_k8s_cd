load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "gitops",
    srcs = ["scan.go"],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/gitops",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
    deps = ["//go/change_report/pkg/label"],
)

go_test(
    name = "gitops_test",
    srcs = ["scan_test.go"],
    embed = [":gitops"],
)

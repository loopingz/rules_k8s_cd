load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "attrib",
    srcs = ["attrib.go"],
    importpath = "loopingz.com/rules_k8s_cd/release/pkg/attrib",
    visibility = ["//go/release:__subpackages__"],
    deps = [
        "//go/change_report/pkg/bazelq",
        "//go/change_report/pkg/cache",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/graph",
        "//go/change_report/pkg/label",
        "//go/change_report/pkg/report",
        "//go/release/pkg/component",
    ],
)

go_test(
    name = "attrib_test",
    srcs = ["attrib_test.go"],
    embed = [":attrib"],
    deps = [
        "//go/change_report/pkg/cache",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/report",
        "//go/release/pkg/component",
    ],
)

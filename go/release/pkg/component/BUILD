load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "component",
    srcs = ["component.go"],
    importpath = "loopingz.com/rules_k8s_cd/release/pkg/component",
    visibility = ["//go/release:__subpackages__"],
    deps = [
        "//go/change_report/pkg/bazelq",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/label",
        "//go/change_report/pkg/report",
        "//go/release/pkg/semver",
        "//go/release/pkg/tagfmt",
    ],
)

go_test(
    name = "component_test",
    srcs = ["component_test.go"],
    data = glob(["testdata/**"]),
    embed = [":component"],
    deps = [
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/report",
        "//go/release/pkg/semver",
    ],
)

load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "release",
    srcs = [
        "pr.go",
        "status.go",
        "tag.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/release/pkg/release",
    visibility = ["//go/release:__subpackages__"],
    deps = [
        "//go/change_report/pkg/bazelq",
        "//go/change_report/pkg/cache",
        "//go/change_report/pkg/gitx",
        "//go/release/pkg/attrib",
        "//go/release/pkg/changelog",
        "//go/release/pkg/component",
        "//go/release/pkg/conventional",
        "//go/release/pkg/forge",
        "//go/release/pkg/gitwrite",
        "//go/release/pkg/semver",
    ],
)

go_test(
    name = "release_test",
    srcs = ["release_test.go"],
    embed = [":release"],
    deps = [
        "//go/change_report/pkg/cache",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/report",
        "//go/release/pkg/attrib",
        "//go/release/pkg/component",
        "//go/release/pkg/forge",
        "//go/release/pkg/semver",
    ],
)

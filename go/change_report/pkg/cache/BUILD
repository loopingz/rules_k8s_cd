load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "cache",
    srcs = [
        "cache.go",
        "keys.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/cache",
    visibility = ["//go/change_report:__subpackages__"],
    deps = [
        "//go/change_report/pkg/bazelq",
        "//go/change_report/pkg/extdeps",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/graph",
        "//go/change_report/pkg/label",
    ],
)

go_test(
    name = "cache_test",
    srcs = ["cache_test.go"],
    embed = [":cache"],
    deps = ["//go/change_report/pkg/gitx"],
)

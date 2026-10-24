load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "extdeps",
    srcs = [
        "diff.go",
        "lockfile.go",
        "showext.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/extdeps",
    visibility = ["//go/change_report:__subpackages__"],
    deps = ["//go/change_report/pkg/bazelq"],
)

go_test(
    name = "extdeps_test",
    srcs = ["extdeps_test.go"],
    data = glob(["testdata/**"]),
    embed = [":extdeps"],
)

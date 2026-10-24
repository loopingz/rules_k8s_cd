load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "render",
    srcs = ["render.go"],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/render",
    visibility = ["//go/change_report:__subpackages__"],
    deps = [
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/report",
    ],
)

go_test(
    name = "render_test",
    srcs = ["render_test.go"],
    data = glob(["testdata/**"]),
    embed = [":render"],
    deps = [
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/report",
    ],
)

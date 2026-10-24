load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "report",
    srcs = [
        "attribute.go",
        "baseline.go",
        "report.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/report",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
    deps = [
        "//go/change_report/pkg/extdeps",
        "//go/change_report/pkg/gitops",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/graph",
        "//go/change_report/pkg/label",
    ],
)

go_test(
    name = "report_test",
    srcs = [
        "baseline_test.go",
        "report_test.go",
    ],
    embed = [":report"],
    deps = [
        "//go/change_report/pkg/gitops",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/graph",
    ],
)

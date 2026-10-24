load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "graph",
    srcs = [
        "graph.go",
        "loads.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/graph",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
    deps = [
        "//go/change_report/pkg/bazelq",
        "//go/change_report/pkg/label",
    ],
)

go_test(
    name = "graph_test",
    srcs = ["graph_test.go"],
    embed = [":graph"],
    deps = ["//go/change_report/pkg/bazelq"],
)

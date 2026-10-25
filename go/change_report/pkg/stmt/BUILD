load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "stmt",
    srcs = ["stmt.go"],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/stmt",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
)

go_test(
    name = "stmt_test",
    srcs = ["stmt_test.go"],
    embed = [":stmt"],
)

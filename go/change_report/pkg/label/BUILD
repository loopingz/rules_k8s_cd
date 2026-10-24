load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "label",
    srcs = ["label.go"],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/label",
    visibility = ["//go/change_report:__subpackages__"],
)

go_test(
    name = "label_test",
    srcs = ["label_test.go"],
    embed = [":label"],
)

load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "loopingz.com/rules_k8s_cd/kyverno_import",
    visibility = ["//visibility:private"],
    deps = [],
)

go_binary(
    name = "kyverno_import",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

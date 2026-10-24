load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "bazelq",
    srcs = ["bazelq.go"],
    importpath = "loopingz.com/rules_k8s_cd/change_report/pkg/bazelq",
    visibility = [
        "//go/change_report:__subpackages__",
        "//go/release:__subpackages__",
    ],
)

go_test(
    name = "bazelq_test",
    srcs = ["bazelq_test.go"],
    data = glob(["testdata/**"]),
    embed = [":bazelq"],
)

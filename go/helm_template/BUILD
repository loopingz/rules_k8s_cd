load("@rules_go//go:def.bzl", "go_binary", "go_library", "go_test")
load("//lib:helm.bzl", "helm_template")

go_library(
    name = "helm_template_lib",
    srcs = ["main.go"],
    importpath = "loopingz.com/rules_k8s_cd/helm_template",
    visibility = ["//visibility:private"],
    deps = ["//go/vendor/gopkg.in/yaml.v3:yaml"],
)

go_binary(
    name = "helm_template",
    embed = [":helm_template_lib"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "helm_template_test",
    srcs = ["main_test.go"],
    embed = [":helm_template_lib"],
    deps = ["//go/vendor/gopkg.in/yaml.v3:yaml"],
)

helm_template(
    name = "rendered",
    chart = "testdata/chart/Chart.yaml",
    data = glob(["testdata/chart/**"]),
    exclude_kinds = ["Secret"],
    exclude_resources = ["Service/prod/*-metrics"],
    include_crds = True,
    kube_version = "1.32.0",
    namespace = "prod",
    release_name = "example",
    set = {"replicas": "7"},
    set_string = {"code": "00123"},
    skip_tests = True,
    values = [
        "testdata/first.yaml",
        "testdata/second.yaml",
    ],
)

genrule(
    name = "chart_archive",
    srcs = glob(["testdata/chart/**"]),
    outs = ["fixture.tgz"],
    cmd = "COPYFILE_DISABLE=1 tar -czhf $@ -C \"$$(dirname $(location testdata/chart/Chart.yaml))/..\" chart",
)

helm_template(
    name = "archived",
    chart = ":chart_archive",
    exclude_kinds = ["Secret"],
    exclude_resources = ["Service/prod/*-metrics"],
    include_crds = True,
    kube_version = "1.32.0",
    namespace = "prod",
    release_name = "example",
    set = {"replicas": "7"},
    set_string = {"code": "00123"},
    skip_tests = True,
    values = [
        "testdata/first.yaml",
        "testdata/second.yaml",
    ],
)

helm_template(
    name = "unfiltered",
    chart = "testdata/chart/Chart.yaml",
    data = glob(["testdata/chart/**"]),
)

go_test(
    name = "helm_template_integration_test",
    srcs = ["integration_test.go"],
    data = [
        ":archived",
        ":rendered",
        ":unfiltered",
    ],
)

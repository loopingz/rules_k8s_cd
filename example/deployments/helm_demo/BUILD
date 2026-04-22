load("@rules_k8s_cd//lib:helm.bzl", "helm_template")

filegroup(
    name = "chart_files",
    srcs = glob(["chart/**/*"]),
)

# NOTE: tagged `manual` because it transitively depends on
# //go/helm_postrender, which does not yet build via Bazel — the vendored
# k8s.io/* dependencies lack generated BUILD.bazel files. Once that
# infrastructure is in place, remove the tag and add a `diff_test` against a
# golden file at `expected.yaml`.
helm_template(
    name = "demo",
    chart = ":chart_files",
    values = ["values.yaml"],
    release_name = "demo",
    namespace = "demo-ns",
    exclude = [
        {"kind": "NetworkPolicy"},
    ],
    patchesStrategicMerge = ["patches/strategic.yaml"],
    patchesJson6902 = ["patches/json6902.yaml"],
    tags = ["manual"],
)

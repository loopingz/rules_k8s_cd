load("@bazel_skylib//rules:diff_test.bzl", "diff_test")
load("//lib:gitops.bzl", "gitops")

# Exercise the gitops macro with an environment exporting to a gitops path.
gitops(
    environments = {
        "test": {
            "cluster": "c",
            "gitops": "gitops/{CLUSTER}/{NAMESPACE}/{PACKAGE}",
            "namespace": "ns",
            "registry": "registry.example.com/test",
        },
    },
)

# First line of the exported manifest, which must name the gitops.test target.
genrule(
    name = "gitops_test_first_line",
    srcs = [":gitops/c/ns/lib/tests/gitops_macro/test.yaml"],
    outs = ["gitops_test_first_line.txt"],
    cmd = "head -n 1 $< > $@",
)

diff_test(
    name = "gitops_macro_header_test",
    file1 = ":gitops_test_first_line",
    file2 = "expected_header.txt",
)

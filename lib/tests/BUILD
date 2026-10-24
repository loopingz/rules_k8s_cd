load("@bazel_skylib//rules:diff_test.bzl", "diff_test")
load("//lib:kustomizer.bzl", "kustomize_gitops")

filegroup(
    name = "kustomize_files",
    srcs = [
        "configmap.yaml",
        "kustomization.yaml",
    ],
)

# kustomize_gitops with the default target: the header names this package's
# "gitops.test" target.
kustomize_gitops(
    name = "gitops.test",
    out = "gitops/test/test.yaml",
    data = [":kustomize_files"],
)

# The rendered (pre-copy) file starts with the rules_k8s_cd:target header.
diff_test(
    name = "gitops_header_test",
    file1 = ":gitops/test/test.yaml",
    file2 = "expected/gitops.test.yaml",
)

# Apart from the header line, the exported file is byte-identical to the
# kustomize output.
genrule(
    name = "gitops_test_without_header",
    srcs = [":gitops/test/test.yaml"],
    outs = ["gitops_test_without_header.yaml"],
    cmd = "tail -n +2 $< > $@",
)

diff_test(
    name = "gitops_header_only_test",
    file1 = ":gitops_test_without_header",
    file2 = ":_gitops.test.kustomize.yaml",
)

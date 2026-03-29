load("@rules_k8s_cd//lib:kyverno.bzl", "kyverno_baseline", "kyverno_best_practices", "kyverno_restricted")
load("//starlark:gitops.bzl", "gitops")

gitops(
    images = {
        "website": "//website:image",
    },
)

kyverno_baseline(
    name = "security_baseline",
    manifests = ["manifests/pod-example.yaml"],
)

kyverno_restricted(
    name = "security_restricted",
    manifests = ["manifests/pod-example.yaml"],
)

kyverno_best_practices(
    name = "best_practices",
    manifests = ["manifests/pod-example.yaml"],
)

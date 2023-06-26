load("@rules_oci//oci:defs.bzl", "oci_image", "oci_image_index", "oci_tarball")
load("@rules_pkg//:pkg.bzl", "pkg_tar")
load("@rules_k8s_cd//starlark:kubectl.bzl", "kubectl", "kubectl_export")
load("@rules_k8s_cd//starlark:utils.bzl", "run_all", "show")
load("//starlark:gitops.bzl", "gitops")
load("@rules_k8s_cd//starlark:grype.bzl", "grype_scan")

filegroup(
    name = "website-src",
    srcs = glob([
        "**/*",
        "!BUILD",
    ]),
)

pkg_tar(
    name = "static_tar",
    srcs = [":website-src"],
    package_dir = "/usr/share/nginx/html",
)

oci_image(
    name = "image",
    base = "@nginx",
    tars = [
        ":static_tar",
    ],
    visibility = ["//visibility:public"],
)

oci_tarball(
    name = "image_tar",
    image = ":image",
    repo_tags = [],
)

# A way to use the grype_scan
grype_scan(
    name = "image_scan",
    srcs = [":image_tar"],
)

kubectl_export(
    name = "kubectl",
    arguments = [
        "version",
        "--client",
    ],
)

# This target will return your contexts
kubectl(
    name = "contexts",
    arguments = [
        "config",
        "get-contexts",
    ],
)

# This target will return empty context
kubectl_export(
    name = "contexts-export",
    arguments = [
        "config",
        "get-contexts",
    ],
    template = "output.txt",
)

show(
    name = "kubeshow",
    src = ":kubectl",
    content = True,
)

run_all(
    name = "run_all",
    targets = [
        ":kubeshow",
        ":kubectl",
        ":contexts",
    ],
)

load("@rules_oci//oci:defs.bzl", "oci_image", "oci_image_index", "oci_tarball")
load("@rules_pkg//:pkg.bzl", "pkg_tar")
load("//starlark:kubectl.bzl", "kubectl", "kubectl_export")
load("//starlark:utils.bzl", "run_all", "show")
load("//starlark:gitops.bzl", "gitops")
load("//starlark:grype.bzl", "grype_scan")

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

grype_scan(
    name = "image_scan",
    srcs = [":image_tar"]

)

kubectl_export(name="kubectl", arguments=["version", "--client"])
kubectl(name="kubectl2", arguments=["version"])

show(name="kubeshow", src=":kubectl", content=True)

run_all(
  name = "run_all",
  targets = [
    ":kubeshow",
    ":kubectl",
    ":kubectl2",
  ]
)
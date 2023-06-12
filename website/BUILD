load("@rules_oci//oci:defs.bzl", "oci_image", "oci_image_index")
load("@rules_pkg//:pkg.bzl", "pkg_tar")
load("//starlark:gitops.bzl", "gitops")

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

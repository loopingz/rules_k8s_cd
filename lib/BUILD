load("@bazel_skylib//:bzl_library.bzl", "bzl_library")

exports_files([
    "oci_push_info.sh.tpl",
    "run_all.sh.tpl",
    "kustomization.yaml.tpl",
    "stamp.sh",
])

# Ensure that users building their own rules can dep on our bzl_library targets for their stardoc
package(default_visibility = ["//visibility:public"])

toolchain_type(
    name = "grype_toolchain_type",
)

toolchain_type(
    name = "dive_toolchain_type",
)

toolchain_type(
    name = "kubectl_toolchain_type",
)

toolchain_type(
    name = "kyverno_toolchain_type",
)

toolchain_type(
    name = "trivy_toolchain_type",
)

toolchain_type(
    name = "buildifier_toolchain_type",
)

bzl_library(
    name = "repositories",
    srcs = ["repositories.bzl"],
    deps = [
        "//lib/private:buildifier_toolchain",
        "//lib/private:dive_toolchain",
        "//lib/private:grype_toolchain",
        "//lib/private:kubectl_toolchain",
        "//lib/private:kyverno_toolchain",
        "//lib/private:trivy_toolchain",
    ],
)

bzl_library(
    name = "oci",
    srcs = ["oci.bzl"],
    deps = [
        "@rules_oci//oci:extensions",
        "@rules_oci//oci:pull",
    ],
)

bzl_library(
    name = "kubectl",
    srcs = ["kubectl.bzl"],
    deps = [
        ":oci",
        ":utils",
        "@aspect_bazel_lib//lib:paths",
        "@aspect_bazel_lib//lib:stamping",
        "@bazel_skylib//lib:dicts",
        "@bazel_skylib//lib:paths",
    ],
)

bzl_library(
    name = "kyverno",
    srcs = ["kyverno.bzl"],
)

bzl_library(
    name = "grype",
    srcs = ["grype.bzl"],
    deps = [
        "//lib/private:scanner_factory",
    ],
)

bzl_library(
    name = "trivy",
    srcs = ["trivy.bzl"],
    deps = [
        "//lib/private:scanner_factory",
    ],
)

bzl_library(
    name = "dive",
    srcs = ["dive.bzl"],
)

bzl_library(
    name = "utils",
    srcs = ["utils.bzl"],
    deps = [
        "@aspect_bazel_lib//lib:repo_utils",
    ],
)

bzl_library(
    name = "kustomizer",
    srcs = ["kustomizer.bzl"],
    deps = [
        ":kubectl",
        ":oci",
        ":utils",
        "@aspect_bazel_lib//lib:stamping",
        "@bazel_skylib//lib:dicts",
    ],
)

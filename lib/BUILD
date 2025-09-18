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
    name = "clair_toolchain_type",
)

toolchain_type(
    name = "trivy_toolchain_type",
)

bzl_library(
    name = "repositories",
    srcs = ["repositories.bzl"],
    deps = [
        "//lib/private:grype_toolchain",
        "//lib/private:dive_toolchain",
        "//lib/private:kubectl_toolchain",
        "//lib/private:kyverno_toolchain",
        "//lib/private:clair_toolchain",
        "//lib/private:trivy_toolchain",
    ],
)
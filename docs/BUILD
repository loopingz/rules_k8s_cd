load("@stardoc//stardoc:stardoc.bzl", "stardoc")

package(default_visibility = ["//visibility:private"])

stardoc(
    name = "kyverno_doc",
    out = "kyverno.md",
    input = "//lib:kyverno.bzl",
    deps = ["//lib:kyverno"],
)

stardoc(
    name = "grype_doc",
    out = "grype.md",
    input = "//lib:grype.bzl",
    deps = ["//lib:grype"],
)

stardoc(
    name = "trivy_doc",
    out = "trivy.md",
    input = "//lib:trivy.bzl",
    deps = ["//lib:trivy"],
)

stardoc(
    name = "dive_doc",
    out = "dive.md",
    input = "//lib:dive.bzl",
    deps = ["//lib:dive"],
)

stardoc(
    name = "utils_doc",
    out = "utils.md",
    input = "//lib:utils.bzl",
    deps = ["//lib:utils"],
)

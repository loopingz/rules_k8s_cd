load("@aspect_bazel_lib//lib:repo_utils.bzl", "repo_utils")

def download_toolchain_binary(rctx, toolchain_name, binary, platform):
    is_windows = platform.startswith("windows_")
    path = rctx.path("bin")

    url, sha256 = binary
    bin = toolchain_name + ".exe" if is_windows else toolchain_name
    if (url.endswith(".tar.gz")):
        bin = toolchain_name + "/" + bin
        rctx.file("BUILD.bazel", """
load("@rules_k8s_cd//lib/private:{name}_toolchain.bzl", "{name}_toolchain")
exports_files(["{bin}"])
{name}_toolchain(name = "{name}_toolchain", bin = "{bin}", visibility = ["//visibility:public"])
""".format(name=toolchain_name, bin=bin))
        rctx.download_and_extract(url, toolchain_name + "/", sha256 = sha256)
    else:
        rctx.file("BUILD.bazel", """
load("@rules_k8s_cd//lib/private:{name}_toolchain.bzl", "{name}_toolchain")
exports_files(["{bin}"])
{name}_toolchain(name = "{name}_toolchain", bin = "{bin}", visibility = ["//visibility:public"])
""".format(name=toolchain_name, bin=bin))
        rctx.download(url = url, output = toolchain_name + ".exe" if is_windows else toolchain_name, sha256 = sha256, executable = True)
"""Pinned Helm toolchains for rendering charts without a local Helm install."""

load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://get.helm.sh/helm-v3.21.4-darwin-amd64.tar.gz", "9173d05edf9592c6be1d0412ffafd935448dfc7a63c2bc732b8c67e55503e8a8"),
    "darwin_arm64": ("https://get.helm.sh/helm-v3.21.4-darwin-arm64.tar.gz", "6e0bf5eb6daafc2b1ec34bb5ba04ef103f3afc16192fb19805c2924d7ea1033f"),
    "linux_amd64": ("https://get.helm.sh/helm-v3.21.4-linux-amd64.tar.gz", "61f88ab166748cb19604d7884cb100ae9ccb13804ddeb98e08af167eacbb6a14"),
    "linux_arm64": ("https://get.helm.sh/helm-v3.21.4-linux-arm64.tar.gz", "b54c04b4e0b2540bbdc08c17a121dab70e9a2ed0de5705528fec68a5fd3b85a7"),
}

DEFAULT_HELM_REPOSITORY = "helm"

HELM_PLATFORMS = {
    "darwin_amd64": struct(
        release_platform = "macos-amd64",
        compatible_with = [
            "@platforms//os:macos",
            "@platforms//cpu:x86_64",
        ],
    ),
    "darwin_arm64": struct(
        release_platform = "macos-arm64",
        compatible_with = [
            "@platforms//os:macos",
            "@platforms//cpu:aarch64",
        ],
    ),
    "linux_amd64": struct(
        release_platform = "linux-amd64",
        compatible_with = [
            "@platforms//os:linux",
            "@platforms//cpu:x86_64",
        ],
    ),
    "linux_arm64": struct(
        release_platform = "linux-arm64",
        compatible_with = [
            "@platforms//os:linux",
            "@platforms//cpu:aarch64",
        ],
    ),
}

_toolchain = create_toolchain(
    name = "helm",
    binaries = _binaries,
    platforms = HELM_PLATFORMS,
)

HelmInfo = _toolchain.info_provider
helm_toolchain = _toolchain.toolchain_rule
helm_toolchains_repo = _toolchain.toolchains_repo
helm_host_alias_repo = _toolchain.host_alias_repo

# Helm archives nest the executable under the release platform directory.
def _helm_platform_repo_impl(rctx):
    url, sha256 = _binaries[rctx.attr.platform]
    rctx.download_and_extract(
        url = url,
        sha256 = sha256,
        stripPrefix = rctx.attr.platform.replace("_", "-"),
    )
    rctx.file("BUILD.bazel", """
load("@rules_k8s_cd//lib/private:helm_toolchain.bzl", "helm_toolchain")
exports_files(["helm"])
helm_toolchain(name = "helm_toolchain", bin = "helm", visibility = ["//visibility:public"])
""")

helm_platform_repo = repository_rule(
    implementation = _helm_platform_repo_impl,
    attrs = {"platform": attr.string(mandatory = True, values = HELM_PLATFORMS.keys())},
)

load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v8.5.1/buildifier-darwin-amd64", "31de189e1a3fe53aa9e8c8f74a0309c325274ad19793393919e1ca65163ca1a4"),
    "darwin_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v8.5.1/buildifier-darwin-arm64", "62836a9667fa0db309b0d91e840f0a3f2813a9c8ea3e44b9cd58187c90bc88ba"),
    "linux_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v8.5.1/buildifier-linux-amd64", "887377fc64d23a850f4d18a077b5db05b19913f4b99b270d193f3c7334b5a9a7"),
    "linux_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v8.5.1/buildifier-linux-arm64", "947bf6700d708026b2057b09bea09abbc3cafc15d9ecea35bb3885c4b09ccd04"),
}

DEFAULT_BUILDIFIER_VERSION = "8.5.1"
DEFAULT_BUILDIFIER_REPOSITORY = "buildifier"

BUILDIFIER_PLATFORMS = {
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
    name = "buildifier",
    binaries = _binaries,
    platforms = BUILDIFIER_PLATFORMS,
)

BuildifierInfo = _toolchain.info_provider
buildifier_toolchain = _toolchain.toolchain_rule
buildifier_toolchains_repo = _toolchain.toolchains_repo
buildifier_platform_repo = _toolchain.platform_repo
buildifier_host_alias_repo = _toolchain.host_alias_repo

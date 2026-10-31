load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.2.0/buildifier-darwin-amd64", "9e78881e91266e3567ed6d590f6c19bb5cd8cfa7aa80c7e1b6e479264d1051b8"),
    "darwin_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.2.0/buildifier-darwin-arm64", "d9c9c3a2826d1d8f592a9d463e62aed7f4dcacc5ab48e8de6be53c420b34b971"),
    "linux_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.2.0/buildifier-linux-amd64", "470844069d710d0b7740e55007071c63e4d00be89a833dce5d1a91790e84dc28"),
    "linux_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.2.0/buildifier-linux-arm64", "e33b41bdb36eca5dfa69647333f4f148272654b98cddcea12f38fac5ec0a00ff"),
}

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

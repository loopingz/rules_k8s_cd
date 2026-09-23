load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.1.0/buildifier-darwin-amd64", "e9e10ff52ec8786fcabccd251c8109ebf31ef7be1f667e27c6e069b96dbdc1f6"),
    "darwin_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.1.0/buildifier-darwin-arm64", "e9804864c407f920f5ecbf03a5e056a8145e11a6ae6b90d2438a3fd106d34473"),
    "linux_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.1.0/buildifier-linux-amd64", "31b6a8aa1e5c746696788f428729701770ad91925873d8256cb885c60e12c77e"),
    "linux_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.1.0/buildifier-linux-arm64", "38d2ed845f560b4a16ddee41de906508a95f8dc85b04e0851b0a71e3a70d3890"),
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

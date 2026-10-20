load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.121.0/grype_0.121.0_darwin_amd64.tar.gz", "27eef17e42f0340af6582cdbc33ad18faf9f601c613ab81d2e1658a94cbc357f"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.121.0/grype_0.121.0_darwin_arm64.tar.gz", "7be39d3a0428c664ce0183e4aba31ed9accffce9eda7e7f6fa84e9c512285f27"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.121.0/grype_0.121.0_linux_amd64.tar.gz", "64e07133d3486115de647a66581461dccf5c732748a0a3119077395565f15519"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.121.0/grype_0.121.0_linux_arm64.tar.gz", "86f570f75ca947ea3dd6486309232bcff6d78e44ba081ae75ab14eab72be3dae"),
}

DEFAULT_GRYPE_REPOSITORY = "grype"

GRYPE_PLATFORMS = {
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
    name = "grype",
    binaries = _binaries,
    platforms = GRYPE_PLATFORMS,
)

GrypeInfo = _toolchain.info_provider
grype_toolchain = _toolchain.toolchain_rule
grype_toolchains_repo = _toolchain.toolchains_repo
grype_platform_repo = _toolchain.platform_repo
grype_host_alias_repo = _toolchain.host_alias_repo

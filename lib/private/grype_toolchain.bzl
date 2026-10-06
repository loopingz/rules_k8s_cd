load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.120.0/grype_0.120.0_darwin_amd64.tar.gz", "e77ed0159ba2c291a7319c2c345cbae79bcd93dee901f77d29472ce1908e5ed0"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.120.0/grype_0.120.0_darwin_arm64.tar.gz", "d27d39ba25ba2d2a1d1568219ff159310d2cb54f3cdd5293e5820c5c784f7f11"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.120.0/grype_0.120.0_linux_amd64.tar.gz", "a5a1218dce63acdac152a6b3b5bb366e7267e36f4069848cf455543b3fa5700e"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.120.0/grype_0.120.0_linux_arm64.tar.gz", "bc0e52b1a0de37e2ff021c4924d689dce7dcff2e7d74b39aea16c0453e69be18"),
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

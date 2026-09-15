load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.118.0/grype_0.118.0_darwin_amd64.tar.gz", "cfeecf3462321c37ec4bd37dcd8a7f6630cc6c0c9997a07ff34002c5d7ef9bb3"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.118.0/grype_0.118.0_darwin_arm64.tar.gz", "938f050bb5076c8aa761867b39843abad2414dfe4cc82b7d36886e634f49c640"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.118.0/grype_0.118.0_linux_amd64.tar.gz", "1d444c5e7360471815f7158f71935fcecc68a3c417d85c7344f770854300bba2"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.118.0/grype_0.118.0_linux_arm64.tar.gz", "32aceeb8ee837244775fcb522372c8b3a47914986385f3148f4ee2c930482a84"),
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

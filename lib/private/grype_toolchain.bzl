load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.119.0/grype_0.119.0_darwin_amd64.tar.gz", "ea106d3ab9573d654871ad9e3e89be2237506ff3f8c170e5aeacb59da4def2b8"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.119.0/grype_0.119.0_darwin_arm64.tar.gz", "500c9b2b6c089d21481815f57a553fabbd441ec7d1e79d95e3aaf40c3bfc7e36"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.119.0/grype_0.119.0_linux_amd64.tar.gz", "3fa2dc4b924621ab65404cf08d0b8438d896d80ab949c9d5a4ca283c36004c9b"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.119.0/grype_0.119.0_linux_arm64.tar.gz", "29f0ec7c549ddb0e2b6a0ca714851f7399438afc399b80c12808e065edc9a8f8"),
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

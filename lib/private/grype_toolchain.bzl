load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.121.1/grype_0.121.1_darwin_amd64.tar.gz", "20fefd09105fb7bf2bbfe29c6bb14edbe4c17214436753735c172465fffe69f6"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.121.1/grype_0.121.1_darwin_arm64.tar.gz", "daa9ffdc0022b34fb476f94cbcabac9d850882c895d83e4a5ecf5e0ab13bce77"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.121.1/grype_0.121.1_linux_amd64.tar.gz", "bc3ea3f9e6fd56ac781c08c6c5e614271b79f438200757f6932085c19b0a4657"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.121.1/grype_0.121.1_linux_arm64.tar.gz", "33670612aad351c48cf3c37f4a85d8b043d1b64b21e6034e675bb00b956d78c1"),
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

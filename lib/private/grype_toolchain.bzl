load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.111.0/grype_0.111.0_darwin_amd64.tar.gz", "8fefd00f6ddd6407275be31b228089820e91c7a8cd2d046e877601773ac5062f"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.111.0/grype_0.111.0_darwin_arm64.tar.gz", "62d005a1e36ac7ec0b7be801ebc8eab0053fd831a227e1dc8ea9c356d38fa361"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.111.0/grype_0.111.0_linux_amd64.tar.gz", "18ed2048d7a233566b681121d4632364f5f25d72cca86acc4c7ac57210d78a87"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.111.0/grype_0.111.0_linux_arm64.tar.gz", "1a8b9bd691ce274e44056e7572cdf8c6970bdf9ec694001f7b4b17962b121b43"),
}

DEFAULT_GRYPE_VERSION = "0.106.0"
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

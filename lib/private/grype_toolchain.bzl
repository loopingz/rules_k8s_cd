load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.110.0/grype_0.110.0_darwin_amd64.tar.gz", "61e767381c395b2fac9ed6016f45de39107761df75f094fc3d9b7822809d07df"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.110.0/grype_0.110.0_darwin_arm64.tar.gz", "9aff01bfcb4510a1b803ef59375b43cd80764fa49aed71f9a3da81c417037411"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.110.0/grype_0.110.0_linux_amd64.tar.gz", "aaa98d27d2d7efd9317c6a1ad6d9b15f3e65bab320e7d03bde41e251387bb71c"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.110.0/grype_0.110.0_linux_arm64.tar.gz", "804041ee69f119022e3e866741a558eae6f2df372a5dc1a5376d456d16f8c931"),
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

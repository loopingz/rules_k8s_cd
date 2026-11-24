load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.122.1/grype_0.122.1_darwin_amd64.tar.gz", "8063aef11b52e6da342068aa6980e92f40b28849eff990c8375a0a8e6b6dfbc5"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.122.1/grype_0.122.1_darwin_arm64.tar.gz", "8bbb3c01695d1e954875cf9f7403c5d3cdb652caf763867955fad520e3cd6c00"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.122.1/grype_0.122.1_linux_amd64.tar.gz", "f1ba7e2917ee1395618de20a98da0aa7c0fb1788927797651047c50502a37538"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.122.1/grype_0.122.1_linux_arm64.tar.gz", "819e5be3d9516432fedb7b6b56df7306c3a16607606dfc05826eefbb00e471da"),
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

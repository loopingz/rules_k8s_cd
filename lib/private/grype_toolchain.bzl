load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.112.0/grype_0.112.0_darwin_amd64.tar.gz", "2fd7862e20ba43589b84919f05a5e6dd3a5b12d3860aed467bc4dc427926f6eb"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.112.0/grype_0.112.0_darwin_arm64.tar.gz", "58c3c372e334c27e5bd5031cfb5ae85dbe5e782478d52fb5515ea413b6d47da4"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.112.0/grype_0.112.0_linux_amd64.tar.gz", "acb14a030010fe9bdb9594b4ae108d9d14ef2f926d936aa0916dc62c89c058ea"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.112.0/grype_0.112.0_linux_arm64.tar.gz", "7fdeccf065965cc59386c656e5fcc1eb1bdf820e2433000bca7f010b8e6da155"),
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

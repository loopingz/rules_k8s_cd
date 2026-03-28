load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/wagoodman/dive/releases/download/v0.13.1/dive_0.13.1_darwin_amd64.tar.gz", "04e4c1bac21be3aef99799cf0e470149a072ea4786be50718aa846cd13746523"),
    "darwin_arm64": ("https://github.com/wagoodman/dive/releases/download/v0.13.1/dive_0.13.1_darwin_arm64.tar.gz", "38b7fa95a13e7f4d0b3060c875fe7427c2a0613ecff674bb45156eb34bca0b09"),
    "linux_amd64": ("https://github.com/wagoodman/dive/releases/download/v0.13.1/dive_0.13.1_linux_amd64.tar.gz", "0970549eb4a306f8825a84145a2534153badb4d7dcf3febd1967c706367c3d0e"),
    "linux_arm64": ("https://github.com/wagoodman/dive/releases/download/v0.13.1/dive_0.13.1_linux_arm64.tar.gz", "2fcd2cf20f634ccdb41efac44048b204bfc867c115641f37a7420693ed480a18"),
}

DEFAULT_DIVE_VERSION = "0.12.0"
DEFAULT_DIVE_REPOSITORY = "dive"

DIVE_PLATFORMS = {
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
    name = "dive",
    binaries = _binaries,
    platforms = DIVE_PLATFORMS,
)

DiveInfo = _toolchain.info_provider
dive_toolchain = _toolchain.toolchain_rule
dive_toolchains_repo = _toolchain.toolchains_repo
dive_platform_repo = _toolchain.platform_repo
dive_host_alias_repo = _toolchain.host_alias_repo

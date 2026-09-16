load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.0.1/buildifier-darwin-amd64", "1d02bb9148cadf2cbee330f9bd657352c765b52b68a03d970e10e47706bdc436"),
    "darwin_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.0.1/buildifier-darwin-arm64", "afb78f350319b59cc51d6add3a5f3ba68e63e5d88f68c5a9ea6328a07084d319"),
    "linux_amd64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.0.1/buildifier-linux-amd64", "e0ea28e2d639347724435ebafe0531fd764fbf20eec6a23000c81edd0d58e51d"),
    "linux_arm64": ("https://github.com/bazelbuild/buildtools/releases/download/v10.0.1/buildifier-linux-arm64", "6d7aebd23aa85847a66d517bb6220d95f24a2752e62cce0f089145b680b539c7"),
}

DEFAULT_BUILDIFIER_REPOSITORY = "buildifier"

BUILDIFIER_PLATFORMS = {
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
    name = "buildifier",
    binaries = _binaries,
    platforms = BUILDIFIER_PLATFORMS,
)

BuildifierInfo = _toolchain.info_provider
buildifier_toolchain = _toolchain.toolchain_rule
buildifier_toolchains_repo = _toolchain.toolchains_repo
buildifier_platform_repo = _toolchain.platform_repo
buildifier_host_alias_repo = _toolchain.host_alias_repo

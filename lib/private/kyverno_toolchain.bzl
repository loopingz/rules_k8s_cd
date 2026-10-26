load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/kyverno/kyverno/releases/download/v1.19.2/kyverno-cli_v1.19.2_darwin_x86_64.tar.gz", "b645654ab8e65bd4aa64de9228e274b388bab19b3404a4e9af70174b3d5ad230"),
    "darwin_arm64": ("https://github.com/kyverno/kyverno/releases/download/v1.19.2/kyverno-cli_v1.19.2_darwin_arm64.tar.gz", "7915c03c613c22363a100adb3da8d06f2e2626d9d21a98b97f921795ffdc536f"),
    "linux_amd64": ("https://github.com/kyverno/kyverno/releases/download/v1.19.2/kyverno-cli_v1.19.2_linux_x86_64.tar.gz", "357eaf1df8bc24ddd5a73979fadeaae6a032e8710f11af264229756e4956bf4a"),
    "linux_arm64": ("https://github.com/kyverno/kyverno/releases/download/v1.19.2/kyverno-cli_v1.19.2_linux_arm64.tar.gz", "250f2a02f143d5c6425e686b13c130d3a3ba7372dba6a7210d599d1b0dd12e92"),
}

DEFAULT_KYVERNO_REPOSITORY = "kyverno"

KYVERNO_PLATFORMS = {
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
    name = "kyverno",
    binaries = _binaries,
    platforms = KYVERNO_PLATFORMS,
)

KyvernoInfo = _toolchain.info_provider
kyverno_toolchain = _toolchain.toolchain_rule
kyverno_toolchains_repo = _toolchain.toolchains_repo
kyverno_platform_repo = _toolchain.platform_repo
kyverno_host_alias_repo = _toolchain.host_alias_repo

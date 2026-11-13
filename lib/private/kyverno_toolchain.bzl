load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/kyverno/kyverno/releases/download/v1.20.0/kyverno-cli_v1.20.0_darwin_x86_64.tar.gz", "2e31ed3df7fda3252a74c897eb6b7b581e634e0759796b43ce0d2472d7376275"),
    "darwin_arm64": ("https://github.com/kyverno/kyverno/releases/download/v1.20.0/kyverno-cli_v1.20.0_darwin_arm64.tar.gz", "9f7e537bdf39ff4f1819d978aec791a4dafa1cfd382f75f53baf422702452149"),
    "linux_amd64": ("https://github.com/kyverno/kyverno/releases/download/v1.20.0/kyverno-cli_v1.20.0_linux_x86_64.tar.gz", "06b8f3ca7b5626473be8fb4d343ad54ab84e317142766f72420e7d44b80942e2"),
    "linux_arm64": ("https://github.com/kyverno/kyverno/releases/download/v1.20.0/kyverno-cli_v1.20.0_linux_arm64.tar.gz", "7d002205369f8ebb8a5570efb6a1bfc4fa02e67ca74d6d8cf2f27783e0e54759"),
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

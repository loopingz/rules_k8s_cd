load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/kyverno/kyverno/releases/download/v1.15.1/kyverno-cli-v1.15.1_darwin_x86_64.tar.gz", "6875b5836f188b089fe4af6d3be8709a61ccad46d7e39febf06472df19d171f5"),
    "darwin_arm64": ("https://github.com/kyverno/kyverno/releases/download/v1.15.1/kyverno-cli_v1.15.1_darwin_arm64.tar.gz", "a6a2a25b1d0ee1ea564cc3303434096f0313f45fdac1ec453b5f63586b2ebdfb"),
    "linux_amd64": ("https://github.com/kyverno/kyverno/releases/download/v1.15.1/kyverno-cli-v1.15.1_linux_x86_64.tar.gz", "6b252750af3063e698f4d72cbf7599e8b292bd710248e23d0b1c8935e88aee67"),
    "linux_arm64": ("https://github.com/kyverno/kyverno/releases/download/v1.15.1/kyverno-cli-v1.15.1_linux_arm64.tar.gz", "de2a9398cd9d75747e0fd50ce824a31389663a0e50e62481ddf8f52a40172d24"),
}

DEFAULT_KYVERNO_VERSION = "1.9.0"
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

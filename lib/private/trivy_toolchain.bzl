load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.0/trivy_0.69.0_macOS-64bit.tar.gz", "4264e4fcc73259de36a68c112a586d65bf6cd488ef2aea857f37d00d8cb5c4e6"),
    "darwin_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.0/trivy_0.69.0_macOS-ARM64.tar.gz", "bd35348d963d3f661ff4d7d138e65a75fedbfade0378689f3a349c824c6e5b75"),
    "linux_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.0/trivy_0.69.0_Linux-ARM64.tar.gz", "425e883f37cad0b512478df2803f58532e7d235267303375a3d0f97e4790a1ca"),
    "linux_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.0/trivy_0.69.0_Linux-64bit.tar.gz", "fff5813d6888fa6f8bd40042a08c4f072b3e65aec9f13dd9ab1d7b26146ad046"),
}

DEFAULT_TRIVY_VERSION = "0.68.2"
DEFAULT_TRIVY_REPOSITORY = "trivy"

TRIVY_PLATFORMS = {
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
    name = "trivy",
    binaries = _binaries,
    platforms = TRIVY_PLATFORMS,
)

TrivyInfo = _toolchain.info_provider
trivy_toolchain = _toolchain.toolchain_rule
trivy_toolchains_repo = _toolchain.toolchains_repo
trivy_platform_repo = _toolchain.platform_repo
trivy_host_alias_repo = _toolchain.host_alias_repo

load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.3/trivy_0.69.3_macOS-64bit.tar.gz", "fec4a9f7569b624dd9d044fca019e5da69e032700edbb1d7318972c448ec2f4e"),
    "darwin_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.3/trivy_0.69.3_macOS-ARM64.tar.gz", "a2f2179afd4f8bb265ca3c7aefb56a666bc4a9a411663bc0f22c3549fbc643a5"),
    "linux_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.3/trivy_0.69.3_Linux-64bit.tar.gz", "1816b632dfe529869c740c0913e36bd1629cb7688bd5634f4a858c1d57c88b75"),
    "linux_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.69.3/trivy_0.69.3_Linux-ARM64.tar.gz", "7e3924a974e912e57b4a99f65ece7931f8079584dae12eb7845024f97087bdfd"),
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

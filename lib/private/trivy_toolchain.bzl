load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.74.0/trivy_0.74.0_macOS-64bit.tar.gz", "472816f6888dda689d075c30254d4210b4d1035acf365aa72332f584c2f60485"),
    "darwin_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.74.0/trivy_0.74.0_macOS-ARM64.tar.gz", "1caada5e0e2091909357c7525d3aa76f4b660b13821bc143b190c7483e31cc11"),
    "linux_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.74.0/trivy_0.74.0_Linux-64bit.tar.gz", "2ae6fe3ee734b7fdf11335663e18c75ea12dccc76062f09f164a3b0f8be4371a"),
    "linux_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.74.0/trivy_0.74.0_Linux-ARM64.tar.gz", "b94ce1976bbf3c15b514b605ee88be7c6d94a29be2302847ff01cb794d47aad5"),
}

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

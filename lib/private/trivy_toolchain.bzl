load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.75.0/trivy_0.75.0_macOS-64bit.tar.gz", "291edaa9778acbe4693d067b5ad60ee11570e5ac68296e85595417528ca641e4"),
    "darwin_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.75.0/trivy_0.75.0_macOS-ARM64.tar.gz", "4a77108cccf8e55c8d6823e1e759939a622277e66cd0daa3c1fc621ed69e4568"),
    "linux_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.75.0/trivy_0.75.0_Linux-64bit.tar.gz", "c6e65abddb348e25f10549df887045629cf28cc72453cd1c63acb717316b3f3f"),
    "linux_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.75.0/trivy_0.75.0_Linux-ARM64.tar.gz", "a1ee9f6ffb7d112b64ff726a2a0717c21175c1114361391f4a132956751a13b3"),
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

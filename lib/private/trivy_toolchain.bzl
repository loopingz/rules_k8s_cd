load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.76.1/trivy_0.76.1_macOS-64bit.tar.gz", "7aa762eacd582052d053b62dcf972a1ab058abe39a57ccae234ed8894945b6b7"),
    "darwin_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.76.1/trivy_0.76.1_macOS-ARM64.tar.gz", "11a9b6cd2752011b54f01ea4bd0dbfa9f0c41dc6538d5a3f9d23955f471bf4e5"),
    "linux_amd64": ("https://github.com/aquasecurity/trivy/releases/download/v0.76.1/trivy_0.76.1_Linux-64bit.tar.gz", "c7cbe3137fb8a48a17574bede3dd6beb16742d1e9772929940fc90082c39d9c0"),
    "linux_arm64": ("https://github.com/aquasecurity/trivy/releases/download/v0.76.1/trivy_0.76.1_Linux-ARM64.tar.gz", "e174dff2a6dba5e3ae41f8d0537a6324405bd08132497340304c6b20e0f16a1f"),
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

load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.120.1/grype_0.120.1_darwin_amd64.tar.gz", "5313004ccbc524c8757521dc3913f1edf4309f56bef06a2fb2d0c0eeade7cc62"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.120.1/grype_0.120.1_darwin_arm64.tar.gz", "cf97957fa467d25575ec2cc3228289f391ea51cf88b9cffc5f83dc03d4cbc732"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.120.1/grype_0.120.1_linux_amd64.tar.gz", "0a9ee97ef5ae2ee953b0a80098105052e846cdbe319a57d808b519c33cd1343d"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.120.1/grype_0.120.1_linux_arm64.tar.gz", "29f47391dc283aa79fcc38e65224cd61f64dec0ecfd0db7074128ebf8ff23514"),
}

DEFAULT_GRYPE_REPOSITORY = "grype"

GRYPE_PLATFORMS = {
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
    name = "grype",
    binaries = _binaries,
    platforms = GRYPE_PLATFORMS,
)

GrypeInfo = _toolchain.info_provider
grype_toolchain = _toolchain.toolchain_rule
grype_toolchains_repo = _toolchain.toolchains_repo
grype_platform_repo = _toolchain.platform_repo
grype_host_alias_repo = _toolchain.host_alias_repo

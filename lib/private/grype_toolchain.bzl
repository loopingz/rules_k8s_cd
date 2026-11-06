load("//lib/private:toolchain_factory.bzl", "create_toolchain")

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.121.2/grype_0.121.2_darwin_amd64.tar.gz", "1a748df28f72e841dea5c6a19c736b61a79c3e81dc376e0c18e1ab0d274cfe08"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.121.2/grype_0.121.2_darwin_arm64.tar.gz", "3633c113d81ec1460b0520e23633d72a25caae62875fd48a8b0e4f22dcc7c704"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.121.2/grype_0.121.2_linux_amd64.tar.gz", "9c8965ced9df657e5e851c14021bae838802f711edd7a48f9bc3c923a8d28f12"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.121.2/grype_0.121.2_linux_arm64.tar.gz", "d7d2e160255b0f087280feabf6f213cf4c45d475690f3f1fe5dc336eb1037bf0"),
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

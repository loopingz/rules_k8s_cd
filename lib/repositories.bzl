load("//lib/private:grype_toolchain.bzl", "GRYPE_PLATFORMS", "grype_platform_repo", "grype_toolchains_repo", "grype_host_alias_repo")
load("//lib/private:dive_toolchain.bzl", "DIVE_PLATFORMS", "dive_platform_repo", "dive_toolchains_repo", "dive_host_alias_repo")
load("//lib/private:kubectl_toolchain.bzl", "KUBECTL_PLATFORMS", "kubectl_platform_repo", "kubectl_toolchains_repo", "kubectl_host_alias_repo")


def register_grype_toolchains(name = "grype", register = True):
    """Registers grype toolchain and repositories

    Args:
        name: override the prefix for the generated toolchain repositories
        register: whether to call through to native.register_toolchains.
            Should be True for WORKSPACE users, but false when used under bzlmod extension
    """
    for [platform, _] in GRYPE_PLATFORMS.items():
        grype_platform_repo(
            name = "%s_%s" % (name, platform),
            platform = platform,
        )

        if register:
            native.register_toolchains("@%s_toolchains//:%s_toolchain" % (name, platform))

    grype_host_alias_repo(name = name)

    grype_toolchains_repo(
        name = "%s_toolchains" % name,
        user_repository_name = name,
    )


def register_dive_toolchains(name = "dive", register = True):
    """Registers grype toolchain and repositories

    Args:
        name: override the prefix for the generated toolchain repositories
        register: whether to call through to native.register_toolchains.
            Should be True for WORKSPACE users, but false when used under bzlmod extension
    """
    for [platform, _] in DIVE_PLATFORMS.items():
        dive_platform_repo(
            name = "%s_%s" % (name, platform),
            platform = platform,
        )

        if register:
            native.register_toolchains("@%s_toolchains//:%s_toolchain" % (name, platform))

    dive_host_alias_repo(name = name)

    dive_toolchains_repo(
        name = "%s_toolchains" % name,
        user_repository_name = name,
    )

def register_kubectl_toolchains(name = "kubectl", register = True):
    """Registers grype toolchain and repositories

    Args:
        name: override the prefix for the generated toolchain repositories
        register: whether to call through to native.register_toolchains.
            Should be True for WORKSPACE users, but false when used under bzlmod extension
    """
    for [platform, _] in KUBECTL_PLATFORMS.items():
        kubectl_platform_repo(
            name = "%s_%s" % (name, platform),
            platform = platform,
        )

        if register:
            native.register_toolchains("@%s_toolchains//:%s_toolchain" % (name, platform))

    kubectl_host_alias_repo(name = name)

    kubectl_toolchains_repo(
        name = "%s_toolchains" % name,
        user_repository_name = name,
    )

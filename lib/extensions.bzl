"Module extensions for use with bzlmod"

load(
    "@rules_k8s_cd//lib:repositories.bzl",
    "register_grype_toolchains",
    "register_dive_toolchains",
    "register_kubectl_toolchains",
)
load("@rules_k8s_cd//lib/private:grype_toolchain.bzl", "DEFAULT_GRYPE_REPOSITORY", "DEFAULT_GRYPE_VERSION")
load("@rules_k8s_cd//lib/private:dive_toolchain.bzl", "DEFAULT_DIVE_REPOSITORY", "DEFAULT_DIVE_VERSION")
load("@rules_k8s_cd//lib/private:kubectl_toolchain.bzl", "DEFAULT_KUBECTL_REPOSITORY", "DEFAULT_KUBECTL_VERSION")
load("@aspect_bazel_lib//lib/private:extension_utils.bzl", "extension_utils")


def _toolchains_extension_impl(mctx):
    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.grype,
        toolchain_name = "grype",
        toolchain_repos_fn = lambda name, version: register_grype_toolchains(name = name, register = False),
    )

    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.dive,
        toolchain_name = "dive",
        toolchain_repos_fn = lambda name, version: register_dive_toolchains(name = name, register = False),
    )

    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.kubectl,
        toolchain_name = "kubectl",
        toolchain_repos_fn = lambda name, version: register_kubectl_toolchains(name = name, register = False),
    )


toolchains = module_extension(
    implementation = _toolchains_extension_impl,
    tag_classes = {
        "grype": tag_class(attrs = {"name": attr.string(default = DEFAULT_GRYPE_REPOSITORY), "version": attr.string(default = DEFAULT_GRYPE_VERSION)}),
        "dive": tag_class(attrs = {"name": attr.string(default = DEFAULT_DIVE_REPOSITORY), "version": attr.string(default = DEFAULT_DIVE_VERSION)}),
        "kubectl": tag_class(attrs = {"name": attr.string(default = DEFAULT_KUBECTL_REPOSITORY), "version": attr.string(default = DEFAULT_KUBECTL_VERSION)}),
    },
)
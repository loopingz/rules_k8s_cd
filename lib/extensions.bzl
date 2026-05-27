"Module extensions for use with bzlmod"

load("@aspect_bazel_lib//lib/private:extension_utils.bzl", "extension_utils")
load(
    "@rules_k8s_cd//lib:repositories.bzl",
    "register_buildifier_toolchains",
    "register_dive_toolchains",
    "register_grype_toolchains",
    "register_kubectl_toolchains",
    "register_kyverno_toolchains",
    "register_trivy_toolchains",
)
load("@rules_k8s_cd//lib/private:buildifier_toolchain.bzl", "DEFAULT_BUILDIFIER_REPOSITORY")
load("@rules_k8s_cd//lib/private:dive_toolchain.bzl", "DEFAULT_DIVE_REPOSITORY")
load("@rules_k8s_cd//lib/private:grype_toolchain.bzl", "DEFAULT_GRYPE_REPOSITORY")
load("@rules_k8s_cd//lib/private:kubectl_toolchain.bzl", "DEFAULT_KUBECTL_REPOSITORY", "DEFAULT_KUBECTL_VERSION")
load("@rules_k8s_cd//lib/private:kyverno_toolchain.bzl", "DEFAULT_KYVERNO_REPOSITORY")
load("@rules_k8s_cd//lib/private:trivy_toolchain.bzl", "DEFAULT_TRIVY_REPOSITORY")

def _toolchains_extension_impl(mctx):
    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.grype,
        toolchain_name = "grype",
        toolchain_repos_fn = lambda name, version: register_grype_toolchains(name = name, register = False),
        get_version_fn = lambda attr: None,
    )

    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.dive,
        toolchain_name = "dive",
        toolchain_repos_fn = lambda name, version: register_dive_toolchains(name = name, register = False),
        get_version_fn = lambda attr: None,
    )

    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.kubectl,
        toolchain_name = "kubectl",
        toolchain_repos_fn = lambda name, version: register_kubectl_toolchains(name = name, register = False),
    )

    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.trivy,
        toolchain_name = "trivy",
        toolchain_repos_fn = lambda name, version: register_trivy_toolchains(name = name, register = False),
        get_version_fn = lambda attr: None,
    )

    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.kyverno,
        toolchain_name = "kyverno",
        toolchain_repos_fn = lambda name, version: register_kyverno_toolchains(name = name, register = False),
        get_version_fn = lambda attr: None,
    )

    extension_utils.toolchain_repos_bfs(
        mctx = mctx,
        get_tag_fn = lambda tags: tags.buildifier,
        toolchain_name = "buildifier",
        toolchain_repos_fn = lambda name, version: register_buildifier_toolchains(name = name, register = False),
        get_version_fn = lambda attr: None,
    )

toolchains = module_extension(
    implementation = _toolchains_extension_impl,
    tag_classes = {
        "grype": tag_class(attrs = {"name": attr.string(default = DEFAULT_GRYPE_REPOSITORY)}),
        "dive": tag_class(attrs = {"name": attr.string(default = DEFAULT_DIVE_REPOSITORY)}),
        "kubectl": tag_class(attrs = {"name": attr.string(default = DEFAULT_KUBECTL_REPOSITORY), "version": attr.string(default = DEFAULT_KUBECTL_VERSION)}),
        "trivy": tag_class(attrs = {"name": attr.string(default = DEFAULT_TRIVY_REPOSITORY)}),
        "kyverno": tag_class(attrs = {"name": attr.string(default = DEFAULT_KYVERNO_REPOSITORY), "policies_dir": attr.string(default = "kyverno/policies")}),
        "buildifier": tag_class(attrs = {"name": attr.string(default = DEFAULT_BUILDIFIER_REPOSITORY)}),
    },
)

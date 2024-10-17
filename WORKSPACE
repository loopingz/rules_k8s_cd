load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Protocol Buffers
# http_archive(
#     name = "com_google_protobuf",
#     sha256 = "e57e9c7dacd6568afd026f4f595191e3a80948fb514911af6ff53a0198c5bfaa",
#     strip_prefix = "protobuf-28.0",
#     urls = [
#         "https://github.com/protocolbuffers/protobuf/archive/refs/tags/v28.0.zip",
#     ],
# )

# load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

# protobuf_deps()

load("@rules_oci//oci:repositories.bzl", "LATEST_CRANE_VERSION", "oci_register_toolchains")

oci_register_toolchains(
    name = "oci",
    crane_version = LATEST_CRANE_VERSION,
)

load("@rules_oci//oci:pull.bzl", "oci_pull")

oci_pull(
    name = "nginx",
    digest = "sha256:367678a80c0be120f67f3adfccc2f408bd2c1319ed98c1975ac88e750d0efe26",
    image = "docker.io/library/nginx",
)

# Adobe rules_gitops
http_archive(
    name = "com_adobe_rules_gitops",
    sha256 = "83124a8056b1e0f555c97adeef77ec6dff387eb3f5bc58f212b376ba06d070dd",
    strip_prefix = "rules_gitops-0.17.2",
    urls = ["https://github.com/adobe/rules_gitops/archive/refs/tags/v0.17.2.tar.gz"],
)

load("@com_adobe_rules_gitops//gitops:deps.bzl", "rules_gitops_dependencies")

rules_gitops_dependencies()

load("@com_adobe_rules_gitops//gitops:repositories.bzl", "rules_gitops_repositories")

rules_gitops_repositories()

load("@com_adobe_rules_gitops//skylib:k8s.bzl", "kubeconfig")

kubeconfig(
    name = "k8s_test",
    cluster = "loopkube",
    use_host_config = True,
)

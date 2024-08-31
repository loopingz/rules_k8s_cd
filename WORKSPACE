load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Rules go
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "67b4d1f517ba73e0a92eb2f57d821f2ddc21f5bc2bd7a231573f11bd8758192e",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.50.0/rules_go-v0.50.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.50.0/rules_go-v0.50.0.zip",
    ],
)

# Download Gazelle.
http_archive(
    name = "bazel_gazelle",
    sha256 = "8ad77552825b078a10ad960bec6ef77d2ff8ec70faef2fd038db713f410f5d87",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.38.0/bazel-gazelle-v0.38.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.38.0/bazel-gazelle-v0.38.0.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

go_rules_dependencies()

go_register_toolchains(version = "1.20.4")

gazelle_dependencies()

# Protocol Buffers
http_archive(
    name = "com_google_protobuf",
    sha256 = "e57e9c7dacd6568afd026f4f595191e3a80948fb514911af6ff53a0198c5bfaa",
    strip_prefix = "protobuf-28.0",
    urls = [
        "https://github.com/protocolbuffers/protobuf/archive/refs/tags/v28.0.zip",
    ],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load("@rules_oci//oci:repositories.bzl", "LATEST_CRANE_VERSION", "oci_register_toolchains")

oci_register_toolchains(
    name = "oci",
    crane_version = LATEST_CRANE_VERSION,
)

load("@rules_oci//oci:pull.bzl", "oci_pull")

oci_pull(
    name = "nginx",
    digest = "sha256:5f0574409b3add89581b96c68afe9e9c7b284651c3a974b6e8bac46bf95e6b7f",
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

load("//starlark:grype.bzl", "grype_setup")

grype_setup()

load("//starlark:kubeconfig.bzl", "kubeconfig")

kubeconfig(
    name = "loopkube",
    cluster = "loopkube",
    use_host_config = True,
)

load("//starlark:kubectl.bzl", "kubectl_setup")

kubectl_setup()

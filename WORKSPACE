load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Rules go
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "d6ab6b57e48c09523e93050f13698f708428cfd5e619252e369d377af6597707",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.43.0/rules_go-v0.43.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.43.0/rules_go-v0.43.0.zip",
    ],
)

# Download Gazelle.
http_archive(
    name = "bazel_gazelle",
    sha256 = "32938bda16e6700063035479063d9d24c60eda8d79fd4739563f50d331cb3209",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.35.0/bazel-gazelle-v0.35.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.35.0/bazel-gazelle-v0.35.0.tar.gz",
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
    sha256 = "eaafa4e19a6619c15df4c30d7213efbfd0f33ad16021cc5f72bbc5d0877346b5",
    strip_prefix = "protobuf-25.1",
    urls = [
        "https://github.com/protocolbuffers/protobuf/archive/refs/tags/v25.1.zip",
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
    digest = "sha256:9784f7985f6fba493ba30fb68419f50484fee8faaf677216cb95826f8491d2e9",
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

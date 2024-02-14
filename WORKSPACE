load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Rules go
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "80a98277ad1311dacd837f9b16db62887702e9f1d1c4c9f796d0121a46c8e184",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.46.0/rules_go-v0.46.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.46.0/rules_go-v0.46.0.zip",
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
    sha256 = "ddd0f5271f31b549efc74eb39061e142132653d5d043071fcec265bd571e73c4",
    strip_prefix = "protobuf-25.2",
    urls = [
        "https://github.com/protocolbuffers/protobuf/archive/refs/tags/v25.2.zip",
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
    digest = "sha256:b41c95c4080d503eac2e455a47280079c5905c6281a1a5ee8fe75b52a92b35a0",
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

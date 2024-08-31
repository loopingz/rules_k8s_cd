load("@rules_k8s_cd//lib:gitops.bzl", _gitops = "gitops")

# Define the different environments
environments = {
    "dev": {
        "namespace": "bazel-dev",
        "cluster": "loopkube",
        "registry": "docker.loopingz.com/bazel-temp",
    },
    "preview": {
        "namespace": "bazel-dev",
        "cluster": "loopkube",
        "registry": "docker.loopingz.com/bazel-temp",
    },
    "beta": {
        "namespace": "bazel-beta",
        "cluster": "loopkube",
        "gitops": "cloud/{CLUSTER}/{NAMESPACE}/{PACKAGE}.yaml",
        "registry": "docker.loopingz.com/bazel",
    },
    "production": {
        "namespace": "bazel-prod",
        "cluster": "gke-loop-1",
        "gitops": "cloud/{CLUSTER}/{NAMESPACE}/{PACKAGE}.yaml",
        "registry": "docker.loopingz.com/bazel",
    },
}

def gitops(images = {}):
    _gitops(images, environments)

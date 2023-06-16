load("//starlark:gitops.bzl", "gitops")
load("//starlark:utils.bzl", "show")
load("//starlark:kubectl.bzl", "kubectl_export", "kustomize_gitops")


gitops(
    images = {
        "website": "//website:image",
    },
)

env = "preview"

filegroup(
    name = "group",
    srcs = glob([
        "manifests/*",
        "manifests/%s/**/*" % env,
        "overlays/%s/**/*.yaml" % env,
        "kustomization.yaml"
    ])
)

kubectl_export(
    name = "kustomize",
    arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", "./deployments/website/"],
    context = [":group"]
)

kustomize_gitops(
    name = "gitops",
    context = [":group"]
)

show(
    name = "show",
    src = ":kustomize",
    content = True
)
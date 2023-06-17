load("//starlark:utils.bzl", "download_binary", "write_source_file", "show", "run_all")
load("//starlark:kubectl.bzl", "kustomization_file", "kustomize_gitops", "kubectl")
load("//starlark:oci.bzl", "image_pushes")

environments = {
    "dev": {
        "namespace": "bazel-dev",
        "cluster": "loopkube",
        "registry": "docker.loopingz.com/bazel-temp",
    },
    "beta": {
        "namespace": "bazel-beta",
        "cluster": "loopkube",
        "gitops": True,
        "registry": "docker.loopingz.com/bazel",
    },
    "production": {
        "namespace": "bazel-prod",
        "cluster": "gke-loop-1",
        "gitops": True,
        "registry": "docker.loopingz.com/bazel",
    },
}

def push_images(images = {}):
    # Push images
    print(images)
    return [
        ":_deploy.dev.kustomization.show",
    ]

def gitops(images = {}):
    name = "deploy"
    for env in environments:
        info = environments[env]
        manifests_target = "_" + name + "." + env + ".manifests"
        kustomization_target = "_" + name + "." + env + ".kustomization"
        package_name = native.package_name().replace("deployments/", "")
        images_pushed = image_pushes(images, info["registry"])

        run_all(
            name = "push.images." + env,
            targets = images_pushed,
        )
        if native.existing_rule(env):
            fail("Macro gitops should only be used once by BUILD file")
        native.filegroup(
            name = manifests_target,
            srcs = native.glob(["manifests/*.yaml", "manifests/%s/**/*.yaml" % env]),
        )
        kustomization_file(
            name = kustomization_target,
            namespace = "bazel-" + env,
            manifests = [":" + manifests_target],
            images = images_pushed,
        )
        show(
            name = kustomization_target + ".show",
            src = ":" + kustomization_target,
            content = True,
        )
        if "gitops" in info and info["gitops"]:
            kustomize_gitops(
                name = "_gitops." + env,
                context = [":" + kustomization_target, ":" + manifests_target],
                export_path = "cloud/{CLUSTER}/{NAMESPACE}/{PACKAGE}.yaml".format(PACKAGE=package_name, CLUSTER=info["cluster"], NAMESPACE=info["namespace"]),
            )
            run_all(
                name = "gitops." + env,
                targets = [
                    ":push.images." + env,
                    ":_gitops." + env,
                ],
                parallel = False,
            )
        else:
            kubectl(
                name = "_deploy." + env,
                chdir = True,
                arguments = ["apply", "--load-restrictor", "LoadRestrictionsNone", "-k", "."],
                context = [":" + kustomization_target, ":" + manifests_target],
            )
            run_all(
                name = "apply." + env,
                targets = [
                    ":push.images." + env,
                    ":_deploy." + env,
                ],
                parallel = False,
            )
            
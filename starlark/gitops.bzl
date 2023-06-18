load("//starlark:utils.bzl", "download_binary", "write_source_file", "show", "run_all")
load("//starlark:kubectl.bzl", "kustomization_file", "kustomize_gitops", "kubectl")
load("//starlark:oci.bzl", "image_pushes")

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
    name = "deploy"
    for env in environments:
        info = environments[env]
        manifests_target = "_" + name + "." + env + ".manifests"
        patches_target = "_" + name + "." + env + ".patches"
        patchesjson6902_target = "_" + name + "." + env + ".patches6902"
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
        native.filegroup(
            name = patches_target,
            srcs = native.glob(["overlays/*.yaml", "overlays/%s/**/*.yaml" % env]),
        )
        native.filegroup(
            name = patchesjson6902_target,
            srcs = native.glob(["patches/*.yaml", "patches/%s/**/*.yaml" % env]),
        )
        kustomization_file(
            name = kustomization_target,
            namespace = "bazel-" + env,
            resources = [":" + manifests_target],
            images = images_pushed,
            commonAnnotations = {
                "commit": "{{STABLE_GIT_COMMIT}}"
            },
            patchesStrategicMerge = [":" + patches_target],
            patchesJson6902 = [":" + patchesjson6902_target]
        )
        context = [":" + kustomization_target, ":" + manifests_target, ":" + patches_target]
        show(
            name = kustomization_target + ".show",
            src = ":" + kustomization_target,
            content = True,
        )
        if "gitops" in info:
            kustomize_gitops(
                name = "_gitops." + env,
                context = context,
                export_path = info["gitops"].format(PACKAGE=package_name, CLUSTER=info["cluster"], NAMESPACE=info["namespace"]),
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
                context = context,
            )
            kubectl(
                name = "_show." + env,
                #chdir = True,
                arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", native.package_name()],
                context = context,
            )
            run_all(
                name = "apply." + env,
                targets = [
                    ":push.images." + env,
                    ":_deploy." + env,
                ],
                parallel = False,
            )
            
load("//starlark:utils.bzl", "download_binary", "run_all", "show", "write_source_file")
load("//starlark:kubectl.bzl", "kubectl", "kustomization_injector", "kustomize_gitops")
load("//starlark:oci.bzl", "image_pushes")
load("@aspect_bazel_lib//lib:expand_template.bzl", "expand_template")

# This method is one way of implementing - feel free to copy/paste and change to your liking
def gitops(images = {}, environments = {}):
    name = "deploy"

    # For each environment we define the different targets
    for env in environments:
        info = environments[env]
        manifests_target = "_" + name + "." + env + ".manifests"
        patches_target = "_" + name + "." + env + ".patches"
        patchesjson6902_target = "_" + name + "." + env + ".patches6902"
        kustomization_target = "_" + name + "." + env + ".kustomization"
        package_name = native.package_name().replace("deployments/", "")

        # Ensure all containers images are pushed to the registry
        images_pushed = image_pushes(images, info["registry"])

        run_all(
            name = "push.images." + env,
            targets = images_pushed,
        )

        # Ensure gitops is only used once per BUILD file
        if native.existing_rule(env):
            fail("Macro gitops should only be used once by BUILD file")

        # Default resources to load
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

        # Expand kustomization model to include the environment specific
        expand_template(
            # A unique name for this target.
            name = kustomization_target + ".tpl",
            # Where to write the expanded file.
            out = "kustomization.yaml." + env + ".tpl",
            # The template file to expand.
            template = Label("//starlark:kustomization.yaml.tpl"),
            substitutions = {
                "{{environment}}": env,
                "{{namespace}}": "bazel-" + env,
                "{{commit}}": "{{STABLE_GIT_COMMIT}}",
                "{{source}}": native.package_name(),
            },
        )

        # Inject our files into the kustomization.yaml file
        kustomization_injector(
            name = kustomization_target,
            input = ":kustomization.yaml." + env + ".tpl",
            resources = [":" + manifests_target],
            images = images_pushed,
            patchesStrategicMerge = [":" + patches_target],
            patchesJson6902 = [":" + patchesjson6902_target],
            repository = info["registry"] + "/",
        )

        # show target is used to debug the kustomization model
        context = [":" + kustomization_target, ":" + manifests_target, ":" + patches_target]
        show(
            name = kustomization_target + ".show",
            src = ":" + kustomization_target,
            content = True,
        )

        # If environment define a gitops file, we export manifests to another folder repository
        if "gitops" in info:
            # Export the manifests to a gitops repository
            kustomize_gitops(
                name = "_gitops." + env,
                context = context,
                export_path = info["gitops"].format(PACKAGE = package_name, CLUSTER = info["cluster"], NAMESPACE = info["namespace"]),
            )

            # Allow one target to generate the manifests and push them to the gitops repository
            run_all(
                name = "gitops." + env,
                targets = [
                    ":push.images." + env,
                    ":_gitops." + env,
                ],
                parallel = False,
            )
        else:
            # Otherwise we apply the manifests directly to the cluster
            kubectl(
                name = "_deploy." + env,
                arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", ".", "|", "{{kubectl}}", "apply", "-f", "-"],
                context = context,
            )
            kubectl(
                name = "_show." + env,
                arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", native.package_name()],
                context = context,
            )

            # Allow one target to push images then apply
            run_all(
                name = "apply." + env,
                targets = [
                    ":push.images." + env,
                    ":_deploy." + env,
                ],
                parallel = False,
            )

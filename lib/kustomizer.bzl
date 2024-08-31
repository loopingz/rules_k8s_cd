load("@aspect_bazel_lib//lib:stamping.bzl", "STAMP_ATTRS", "maybe_stamp")
load(":kubectl.bzl", "kubectl", "kubectl_export")
load(":utils.bzl", "write_source_file")
load(":oci.bzl", "ContainerPushInfo")
load("@bazel_skylib//lib:dicts.bzl", "dicts")
load("@bazel_skylib//lib:paths.bzl", "paths")

def kustomize(name, data = [], template = "", **kwargs):
    if (template == ""):
        template = name + ".yaml"
    kubectl_export(
        name = name,
        arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", native.package_name() + "/"],
        data = data,
        template = template,
        **kwargs
    )

# Generates a kustomize command for the given Kustomize resource and shows the output.
# If a template is not provided, it defaults to the resource name with a .yaml extension.
#
# name: The name of the Kubernetes resource to generate the kustomization file for.
# data: The Kubernetes context to use. Defaults to the current context.
# template: The name of the template file to use. Defaults to the resource name with a .yaml extension.
# kwargs: Additional arguments to pass to the kubectl command.
def kustomize_show(name, data = [], **kwargs):
    kubectl(
        name = name,
        arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", native.package_name()],
        data = data,
        **kwargs
    )

# Generates a kustomize command for the given Kustomize resource and apply it.
# If a template is not provided, it defaults to the resource name with a .yaml extension.
#
# name: The name of the Kubernetes resource to generate the kustomization file for.
# context: The Kubernetes context to use.
# data: The Kustomize directory to use1.
# template: The name of the template file to use. Defaults to the resource name with a .yaml extension.
# kwargs: Additional arguments to pass to the kubectl command.
def kustomize_apply(name, context = None, data = [], **kwargs):
    args = ["kustomize"]
    if context != None:
        args.extend(["--context", context])
    args.extend(["--load-restrictor", "LoadRestrictionsNone", native.package_name(), "|", "{{kubectl}}"])
    if context != None:
        args.extend(["--context", context])
    args.extend(["apply", "-f", "-"])
    kubectl(
        name = name,
        arguments = args,
        data = data,
        **kwargs
    )

# Creates a kustomization for GitOps deployment using kubectl.
# And export its result to a file in the given path.
# 
# Args:
# - name (str): The name of the kustomization.
# - context (List[Label]): The context to use for kubectl.
# - export_path (str): The path to export the kustomization to.
#
# Returns:
# - None
def kustomize_gitops(name, data, export_path):
    # Generate the yaml file
    kustomize(
        name = "_" + name + ".kustomize",
        data = data,
        out = paths.basename(export_path),
        visibility = ["//visibility:private"],
    )
    # Copying the yaml file to the export path
    write_source_file(
        name = name,
        src = ":_" + name + ".kustomize",
        target = paths.dirname(export_path),
    )

# Implementation of injector
#  - preparing inputs/outputs for the go binary //go/kustomizer:kustomizer
def _kustomization_injector_impl(ctx):
    out = ctx.actions.declare_file("kustomization.yaml")
    builddir = ctx.build_file_path.split("/")
    builddir.pop()
    builddir = "/".join(builddir) + "/"
    arguments = [
        "--input=%s" % ctx.files.input[0].path,
        "--output=%s" % out.path,
        "--relativePath=%s" % builddir,
    ]
    for img in ctx.attr.images:
        arguments.append("--image=%s:oci_push_info://%s" % (img[ContainerPushInfo].name, img.files.to_list()[0].path))
    for img in ctx.attr.external_images:
        arguments.append("--image=%s:ref://%s" % (img, ctx.attr.external_images[img]))
    for res in ctx.files.resources:
        arguments.append("--path=resources:%s" % res.path)
    for res in ctx.files.crds:
        arguments.append("--path=crds:%s" % res.path)
    for res in ctx.files.configMapGenerator:
        arguments.append("--path=configMapGenerator:%s" % res.path)
    for res in ctx.files.secretGenerator:
        arguments.append("--path=secretGenerator:%s" % res.path)
    for res in ctx.files.patchesStrategicMerge:
        arguments.append("--path=patchesStrategicMerge:%s" % res.path)
    for res in ctx.files.patchesJson6902:
        arguments.append("--path=patchesJson6902:%s" % res.path)
    for res in ctx.attr.substitutions:
        arguments.append("--var=%s:%s" % (res, ctx.attr.substitutions[res]))

    if (ctx.attr.repository != ""):
        arguments.append("--repository=%s" % ctx.attr.repository)

    inputs = ctx.files.input + ctx.files.images + ctx.files.resources + ctx.files.crds + ctx.files.configMapGenerator + ctx.files.secretGenerator + ctx.files.patchesStrategicMerge + ctx.files.patchesJson6902
    # Add stamps to substitution
    stamp = maybe_stamp(ctx)
    if stamp:
        arguments.append("--path=stamp:%s" % stamp.volatile_status_file.path)
        arguments.append("--path=stamp:%s" % stamp.stable_status_file.path)
        inputs = inputs + [stamp.volatile_status_file, stamp.stable_status_file]
    
    if ctx.outputs.combine:
        arguments.append("--combine=%s" % ctx.outputs.combine.path)

    ctx.actions.run(
        executable = ctx.executable._kustomizer,
        arguments = arguments,
        outputs = [out, ctx.outputs.combine] if ctx.outputs.combine else [out],
        inputs = inputs,
    )
    return [DefaultInfo(files = depset([out, ctx.outputs.combine] if ctx.outputs.combine else [out]))]

# Rule to inject images, resources, patches, config maps, secrets and substitutions into a kustomization file.
# 
# Args:
# - input (label, mandatory): Input kustomization file.
# - combine (output): Output combined manifest files into a single file and replace variables.
# - repository (string, optional): Images repository to use as prefix for images.
# - images (label_list, optional): List of images to inject in the kustomization file.
# - resources (label_list, optional): List of resources to inject in the kustomization file.
# - patchesStrategicMerge (label_list, optional): List of patches to inject in the kustomization file.
# - patchesJson6902 (label_list, optional): List of patches to inject in the kustomization file.
# - crds (label_list, optional): List of patches to inject in the kustomization file.
# - configMapGenerator (label_list, optional): List of patches to inject in the kustomization file.
# - secretGenerator (label_list, optional): List of secrets to inject in the kustomization file.
# - substitutions (string_dict, optional): Replace variables within the kustomization file (after all other operations).
#
# Returns:
# - kustomization.yaml template and output file with the combined manifest files into a single file and replaced variables.
kustomization_injector = rule(
    implementation = _kustomization_injector_impl,
    attrs = dicts.add({
        "input": attr.label(allow_single_file = True, mandatory = True, doc = "Input kustomization file"),
        "combine": attr.output(doc = "Output combined manifest files into a single file and replace variables"),
        "repository": attr.string(default = "", doc = "Images repository to use as prefix for images"),
        "images": attr.label_list(
            providers = [ContainerPushInfo],
            allow_files = True,
            doc = "List of images to inject in the kustomization file",
        ),
        "external_images": attr.string_dict(
            doc = "List of images to inject in the kustomization file",
        ),
        "resources": attr.label_list(
            allow_files = True,
            doc = "List of resources to inject in the kustomization file",
        ),
        "patchesStrategicMerge": attr.label_list(
            allow_files = True,
            doc = "List of patches to inject in the kustomization file",
        ),
        "patchesJson6902": attr.label_list(
            allow_files = True,
            doc = "List of patches to inject in the kustomization file",
        ),
        "crds": attr.label_list(
            allow_files = True,
            doc = "List of patches to inject in the kustomization file",
        ),
        "configMapGenerator": attr.label_list(
            allow_files = True,
            doc = "List of patches to inject in the kustomization file",
        ),
        "secretGenerator": attr.label_list(
            allow_files = True,
            doc = "List of secrets to inject in the kustomization file",
        ),
        "substitutions": attr.string_dict(
            doc = "Replace variables within the kustomization file (after all other operations)"
        ),
        "_kustomizer": attr.label(
            default = Label("//go/kustomizer:kustomizer"),
            cfg = "exec",
            executable = True,
            allow_files = True,
        ),
    }, **STAMP_ATTRS),
)

load("//starlark:utils.bzl", "download_binary", "run_all", "show", "write_source_file")
load("//starlark:oci.bzl", "ContainerPushInfo")
load("@aspect_bazel_lib//lib:stamping.bzl", "STAMP_ATTRS", "maybe_stamp")
load("@aspect_bazel_lib//lib:paths.bzl", "relative_file")
load("@bazel_skylib//lib:dicts.bzl", "dicts")
load("@bazel_skylib//lib:paths.bzl", "paths")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl.sha256

_binaries = {
    "darwin_arm64": ("https://dl.k8s.io/release/v1.27.2/bin/darwin/arm64/kubectl", "d2b045b1a0804d4c46f646aeb6dcd278202b9da12c773d5e462b1b857d1f37d7"),
    "darwin_amd64": ("https://dl.k8s.io/release/v1.27.2/bin/darwin/amd64/kubectl", "ec954c580e4f50b5a8aa9e29132374ce54390578d6e95f7ad0b5d528cb025f85"),
    "linux_amd64": ("https://dl.k8s.io/release/v1.27.2/bin/linux/amd64/kubectl", "4f38ee903f35b300d3b005a9c6bfb9a46a57f92e89ae602ef9c129b91dc6c5a5"),
    "linux_arm64": ("https://dl.k8s.io/release/v1.27.2/bin/linux/arm64/kubectl", "1b0966692e398efe71fe59f913eaec44ffd4468cc1acd00bf91c29fa8ff8f578"),
}

# FILEPATH: /Users/loopingz/Git/rules_k8s_cd/starlark/kubectl.bzl
# Sets up kubectl binary by downloading it if it is not already present.
#
# Args:
#   name (str): The name of the kubectl binary.
#   binaries (dict): A dictionary containing the URLs and SHA256 hashes of the kubectl binaries.
#   bin (str): The name of the binary file to be downloaded.
#
# Returns:
#   None
def kubectl_setup(name = "kubectl_bin", binaries = _binaries, bin = ""):
    if (bin == ""):
        bin = name.replace("_bin", "")
    download_binary(name = name, binaries = binaries, bin = bin)

def _kubectl_impl(ctx):
    inputs = []
    for f in ctx.attr.context:
        inputs = inputs + f.files.to_list()

    command = ""
    launch = ctx.outputs.launch
    args = [ctx.executable._kubectl.short_path] + ctx.attr.arguments
    for i in range(len(args)):
        if args[i] == "{{kubectl}}":
            args[i] = ctx.executable._kubectl.short_path
    
    for f in ctx.files.data:
        p = f.path
        if p.startswith("bazel-out"):
            src = p
            dst = p[p.index("/bin/") + 5:]
            rel_src = relative_file(src, dst)
            command += "[ ! -f \"%s\"  ] && mkdir -p `dirname %s` && ln -s %s %s\n" % (dst, dst, rel_src, dst)

    command += " ".join(args)

    ctx.actions.write(
        output = ctx.outputs.launch,
        content = command,
        is_executable = True,
    )

    return [DefaultInfo(
        executable = ctx.outputs.launch,
        runfiles = ctx.runfiles(files = [
            ctx.executable._kubectl,
        ] + inputs),
    )]

# Defines a Starlark rule named "kubectl" that generates a shell script to launch kubectl command.
# The rule takes the following attributes:
# - "arguments": a list of strings representing the arguments to pass to the kubectl command.
# - "context": a list of labels representing the Kubernetes contexts to use.
#
# The rule generates a shell script named "{name}.sh" as output.
# The rule is executable and cannot be used for testing.
kubectl = rule(
    implementation = _kubectl_impl,
    attrs = {
        "arguments": attr.string_list(),
        "data": attr.label_list(),
        "_kubectl": attr.label(
            cfg = "host",
            executable = True,
            default = Label("@kubectl_bin//:kubectl_bin"),
        ),
    },
    outputs = {"launch": "%{name}.sh"},
    test = False,
    executable = True,
)


# Implementation of kubectl export
# Creating the resources and capturing the stdout with a comment
def _kubectl_export_impl(ctx):
    launch = ctx.actions.declare_file(ctx.attr.name + ".sh")

    command = ""
    output = ctx.outputs.out.path
    args = [ctx.executable._kubectl.path] + ctx.attr.arguments

    for f in ctx.files.data:
        p = f.path
        if p.startswith("bazel-out"):
            src = p
            dst = p[p.index("/bin/") + 5:]
            rel_src = relative_file(src, dst)
            command += "mkdir -p `dirname %s` && ln -s %s %s\n" % (dst, rel_src, dst)

    command += "echo '# Generated with rules_k8s_cd from bazel build //%s' > %s\n" % (paths.dirname(ctx.build_file_path) + ":" + ctx.attr.name, output)
    command += " ".join(args)

    command += " >> %s" % (output)

    ctx.actions.write(
        output = launch,
        content = command,
        is_executable = True,
    )
    inputs = []
    for f in ctx.attr.context:
        inputs = inputs + f.files.to_list()
    ctx.actions.run(
        executable = launch,
        outputs = [ctx.outputs.out],
        inputs = inputs,
        tools = [ctx.executable._kubectl],
    )

# Defines a Starlark rule named "kubectl_export" that generates a Kubernetes YAML file
# by running the "kubectl" command with the specified arguments and context.
#
# Can be used at build to generate resources
# 
# The rule has the following attributes:
# - "arguments": a list of strings representing the arguments to pass to the "kubectl" command.
# - "template": an output file representing the template file to use for generating the YAML file.
# - "context": a list of labels representing the Kubernetes contexts to use for running the "kubectl" command.
kubectl_export = rule(
    implementation = _kubectl_export_impl,
    attrs = {
        "arguments": attr.string_list(),
        "out": attr.output(),
        "data": attr.label_list(),
        "_kubectl": attr.label(
            cfg = "host",
            executable = True,
            default = Label("@kubectl_bin//:kubectl_bin"),
        ),
    },
    test = False,
    executable = False,
)

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
# context: The Kubernetes context to use. Defaults to the current context.
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
def kustomize_apply(name, context, data = [], **kwargs):
    kubectl(
        name = name,
        arguments = ["kustomize", "--context", context, "--load-restrictor", "LoadRestrictionsNone", native.package_name(), "|", "{{kubectl}}", "--context", context,"apply", "-f", "-"],
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

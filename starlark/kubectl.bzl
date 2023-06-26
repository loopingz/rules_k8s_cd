load("//starlark:utils.bzl", "download_binary", "run_all", "show", "write_source_file")
load("//starlark:oci.bzl", "ContainerPushInfo")
load("@aspect_bazel_lib//lib:stamping.bzl", "STAMP_ATTRS", "maybe_stamp")
load("@aspect_bazel_lib//lib:paths.bzl", "relative_file")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl.sha256

_binaries = {
    "darwin_arm64": ("https://dl.k8s.io/release/v1.27.2/bin/darwin/arm64/kubectl", "d2b045b1a0804d4c46f646aeb6dcd278202b9da12c773d5e462b1b857d1f37d7"),
    "darwin_amd64": ("https://dl.k8s.io/release/v1.27.2/bin/darwin/amd64/kubectl", "ec954c580e4f50b5a8aa9e29132374ce54390578d6e95f7ad0b5d528cb025f85"),
    "linux_amd64": ("https://dl.k8s.io/release/v1.27.2/bin/linux/amd64/kubectl", "4f38ee903f35b300d3b005a9c6bfb9a46a57f92e89ae602ef9c129b91dc6c5a5"),
    "linux_arm64": ("https://dl.k8s.io/release/v1.27.2/bin/linux/amd64/kubectl", "1b0966692e398efe71fe59f913eaec44ffd4468cc1acd00bf91c29fa8ff8f578"),
}

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
    
    for f in ctx.files.context:
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

kubectl = rule(
    implementation = _kubectl_impl,
    attrs = {
        "arguments": attr.string_list(),
        "context": attr.label_list(),
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

def _kubectl_export_impl(ctx):
    launch = ctx.actions.declare_file(ctx.attr.name + ".sh")

    # Export target name
    paths = ctx.build_file_path.split("/")
    paths.pop()
    command = ""
    output = ctx.outputs.template.path
    args = [ctx.executable._kubectl.path] + ctx.attr.arguments

    for f in ctx.files.context:
        p = f.path
        if p.startswith("bazel-out"):
            src = p
            dst = p[p.index("/bin/") + 5:]
            rel_src = relative_file(src, dst)
            command += "mkdir -p `dirname %s` && ln -s %s %s\n" % (dst, rel_src, dst)

    command += "echo '# Generated from bazel build //%s' > %s\n" % ("/".join(paths) + ":" + ctx.attr.name, output)
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
        outputs = [ctx.outputs.template],
        inputs = inputs,
        tools = [ctx.executable._kubectl],
    )

kubectl_export = rule(
    implementation = _kubectl_export_impl,
    attrs = {
        "arguments": attr.string_list(),
        "template": attr.output(),
        "context": attr.label_list(),
        "_kubectl": attr.label(
            cfg = "host",
            executable = True,
            default = Label("@kubectl_bin//:kubectl_bin"),
        ),
    },
    test = False,
    executable = False,
)

def kustomize(name, context = [], template = "", **kwargs):
    if (template == ""):
        template = name + ".yaml"
    kubectl_export(
        name = name,
        arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", native.package_name() + "/"],
        context = context,
        template = template,
        **kwargs
    )

def kustomize_gitops(name, context = [], export_path = "cloud/{CLUSTER}/{NAMESPACE}/", template = ""):
    kustomize(
        name = "_" + name + ".kustomize",
        context = context,
        template = template,
        visibility = ["//visibility:private"],
    )
    write_source_file(
        name = name,
        src = ":_" + name + ".kustomize",
        target = export_path.format(CLUSTER = "loopkube", NAMESPACE = "bazel-test"),
    )

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

    if (ctx.attr.repository != ""):
        arguments.append("--repository=%s" % ctx.attr.repository)

    ctx.actions.run(
        executable = ctx.executable._kustomizer,
        arguments = arguments,
        outputs = [out],
        inputs = ctx.files.input + ctx.files.images + ctx.files.resources + ctx.files.crds + ctx.files.configMapGenerator + ctx.files.secretGenerator + ctx.files.patchesStrategicMerge + ctx.files.patchesJson6902,
    )
    return [DefaultInfo(files = depset([out]))]

kustomization_injector = rule(
    implementation = _kustomization_injector_impl,
    attrs = dict({
        "input": attr.label(allow_single_file = True, mandatory = True, doc = "Input kustomization file"),
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
        "_kustomizer": attr.label(
            default = Label("//go/kustomizer:kustomizer"),
            cfg = "exec",
            executable = True,
            allow_files = True,
        ),
    }),
)

load("//starlark:utils.bzl", "download_binary", "write_source_file", "show", "run_all")
load("//starlark:oci.bzl", "ContainerPushInfo")

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
    command = ""
    launch = ctx.outputs.launch
    if ctx.attr.chdir:
        command += "cd %s\n" % launch.dirname
        output = ctx.outputs.template.basename
        upupup = "/".join([".."] * (launch.dirname.count("/") + 1))
        args = [upupup + "/" + ctx.executable._kubectl.path] + ctx.attr.arguments
    else:
        args = [ctx.executable._kubectl.path] + ctx.attr.arguments

    command = " ".join(args)

    ctx.actions.write(
        output = ctx.outputs.launch,
        content = command,
        is_executable = True,
    )

    return [DefaultInfo(
        executable = ctx.outputs.launch,
        runfiles = ctx.runfiles(files = [
            ctx.executable._kubectl,
        ])
    )]

kubectl = rule(
    implementation = _kubectl_impl,
    attrs = {
        "arguments": attr.string_list(),
        "context": attr.label_list(),
        "chdir": attr.bool(default = False),
        "_kubectl": attr.label(
            cfg = "host",
            executable = True,
            default = Label("@kubectl_bin//:kubectl_bin"),
        )
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
    if ctx.attr.chdir:
        command += "cd %s\n" % launch.dirname
        output = ctx.outputs.template.basename
        upupup = "/".join([".."] * (launch.dirname.count("/") + 1))
        args = [upupup + "/" + ctx.executable._kubectl.path] + ctx.attr.arguments
    else:
        args = [ctx.executable._kubectl.path] + ctx.attr.arguments
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
        "chdir": attr.bool(default = False),
        "_kubectl": attr.label(
            cfg = "host",
            executable = True,
            default = Label("@kubectl_bin//:kubectl_bin"),
        )
    },
    test = False,
    executable = False,
)

def kustomize(name, context = [], template = "", **kwargs):
    if (template == ""):
        template = name + ".yaml"
    kubectl_export(
        name = name,
        chdir = True,
        arguments = ["kustomize", "--load-restrictor", "LoadRestrictionsNone", "."],
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
        target = export_path.format(CLUSTER="loopkube", NAMESPACE="bazel-test"),
    )


kustomization_yaml = """
namespace: {{NAMESPACE}}
namePrefix: {{PREFIX}}-
nameSuffix: {{SUFFIX}}
commonLabels:
  app: {{APP}}
  env: {{ENV}}
  version: {{VERSION}}
commonAnnotations:
    app: {{APP}}
    env: {{ENV}}
    version: {{VERSION}}
resources:
 - file
 - file
configMapGenerator:
- name: {{APP}}-config
  files:
  - config.yaml
secretGenerator:
- name: {{APP}}-secret
  files:
  - secret.yaml
generatorOptions:
    disableNameSuffixHash: true
bases:
 -
 -
patchesStrategicMerge:
    - patch.yaml
    - patch.yaml
patchesJson6902:
    - patch.yaml
    - patch.yaml
vars:
    -  
images:

# configurations: Not supported yet
crds:
 - 
 - 
"""
def _kustomization_file_impl(ctx):
    partial_out = ctx.actions.declare_file("partial_kustomization.yaml")
    out = ctx.actions.declare_file("kustomization.yaml")
    image_injector = ctx.actions.declare_file("image_injector.sh")
    root = out.dirname
    # hack to get relative path to glob resources...
    upupup = "/".join([".."] * (root.count("/") + 1))

    content = ""
    if (ctx.attr.namespace != ""):
        content += "namespace: %s\n" % ctx.attr.namespace
    if (ctx.attr.namePrefix != ""):
        content += "namePrefix: %s\n" % ctx.attr.namePrefix
    if (ctx.attr.nameSuffix != ""):
        content += "nameSuffix: %s\n" % ctx.attr.nameSuffix
    content += "resources:\n"
    for m in ctx.files.manifests:
        #for f in m.files.to_list():
        content += " - %s/%s\n" % (upupup, m.path)

    injections = "cat %s >> %s\necho \"images:\n\" >> %s" % (partial_out.path,out.path,out.path)
    injections_inputs = [
        partial_out
    ]
    for img in ctx.attr.images:
        injections += "echo \" - name: %s\n    newName: %s@$(cat %s)\n\" >> %s" % (img[ContainerPushInfo].name, img[ContainerPushInfo].registry + "/" + img[ContainerPushInfo].repository, img[ContainerPushInfo].digestfile.path, out.path)
        injections_inputs.append(img[ContainerPushInfo].digestfile)

    # Write kustomization_file
    ctx.actions.write(
        output = partial_out,
        content = content,
    )
    ctx.actions.write(
        output = image_injector,
        content = injections,
        is_executable = True,
    )
    ctx.actions.run(
        executable = image_injector,
        outputs = [out],
        inputs = injections_inputs,
    )
    return [DefaultInfo(files = depset([out]))]

kustomization_file = rule(
    implementation = _kustomization_file_impl,
    attrs = {
        "namespace": attr.string(),
        "namePrefix": attr.string(),
        "nameSuffix": attr.string(),
        "commonLabels": attr.string_dict(),
        "commonAnnotations": attr.string_dict(),
        "manifests": attr.label_list(),
        "images": attr.label_list(providers = [ContainerPushInfo])
    },
)

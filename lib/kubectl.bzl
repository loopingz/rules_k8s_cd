load("@aspect_bazel_lib//lib:paths.bzl", "relative_file")
load("@aspect_bazel_lib//lib:stamping.bzl", "STAMP_ATTRS", "maybe_stamp")
load("@bazel_skylib//lib:dicts.bzl", "dicts")
load("@bazel_skylib//lib:paths.bzl", "paths")
load(":oci.bzl", "ContainerPushInfo")
load(":utils.bzl", "download_binary", "run_all", "show", "write_source_file")

def _kubectl_impl(ctx):
    inputs = []
    for f in ctx.attr.data:
        inputs = inputs + f.files.to_list()
    kubectl_bin = ctx.toolchains["@rules_k8s_cd//lib:kubectl_toolchain_type"].kubectlinfo.bin
    command = ""
    launch = ctx.outputs.launch
    args = [kubectl_bin.short_path] + ctx.attr.arguments
    for i in range(len(args)):
        if args[i] == "{{kubectl}}":
            args[i] = kubectl_bin.path

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
            kubectl_bin,
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
        "data": attr.label_list(allow_files = True),
    },
    toolchains = ["@rules_k8s_cd//lib:kubectl_toolchain_type"],
    outputs = {"launch": "%{name}.sh"},
    test = False,
    executable = True,
)

# Implementation of kubectl export
# Creating the resources and capturing the stdout with a comment
def _kubectl_export_impl(ctx):
    launch = ctx.actions.declare_file(ctx.attr.name + ".sh")
    kubectl_bin = ctx.toolchains["@rules_k8s_cd//lib:kubectl_toolchain_type"].kubectlinfo.bin
    command = ""
    output = ctx.outputs.out.path
    args = [kubectl_bin.path] + ctx.attr.arguments

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
    for f in ctx.attr.data:
        inputs = inputs + f.files.to_list()
    ctx.actions.run(
        executable = launch,
        outputs = [ctx.outputs.out],
        inputs = inputs,
        tools = [kubectl_bin],
    )

# Defines a Starlark rule named "kubectl_export" that generates a Kubernetes YAML file
# by running the "kubectl" command with the specified arguments and context.
#
# Can be used at build to generate resources
#
# The rule has the following attributes:
# - "arguments": a list of strings representing the arguments to pass to the "kubectl" command.
# - "out": an output file representing the template file to use for generating the YAML file.
# - "context": a list of labels representing the Kubernetes contexts to use for running the "kubectl" command.
kubectl_export = rule(
    implementation = _kubectl_export_impl,
    attrs = {
        "arguments": attr.string_list(),
        "out": attr.output(),
        "data": attr.label_list(),
    },
    toolchains = ["@rules_k8s_cd//lib:kubectl_toolchain_type"],
    test = False,
    executable = False,
)

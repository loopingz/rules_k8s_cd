def _kyverno_impl(ctx):
    kyverno_bin = ctx.toolchains["@rules_k8s_cd//lib:kyverno_toolchain_type"].kyvernoinfo.bin
    cmd = ""
    command = [kyverno_bin.short_path, "apply"]
    args = []
    for f in ctx.files.manifests:
        args.append("-r")
        args.append(f.short_path)

    cmd += " ".join(command + [f.short_path for f in ctx.files.policies] + args)

    # Write the file that will be executed by 'bazel test'.
    ctx.actions.write(
        output = ctx.outputs.test,
        content = cmd,
    )

    return [DefaultInfo(
        executable = ctx.outputs.test,
        runfiles = ctx.runfiles(files = [
            kyverno_bin,
        ] + ctx.files.policies + ctx.files.manifests),
    )]

# Rule that validates Kubernetes manifests against Kyverno policies.
kyverno_test = rule(
    implementation = _kyverno_impl,
    attrs = {
        "policies": attr.label_list(
            mandatory = True,
            allow_files = [".yaml"],
        ),
        "manifests": attr.label_list(
            mandatory = True,
            #allow_files = [".yaml"],
            doc = ("List of manifests. The test will scan all images defined inside manifests."),
        ),
    },
    toolchains = ["@rules_k8s_cd//lib:kyverno_toolchain_type"],
    outputs = {"test": "%{name}.sh"},
    test = True,
    executable = False,
)

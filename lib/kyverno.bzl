def _kyverno_impl(ctx):
    kyverno_bin = ctx.toolchains["@rules_k8s_cd//lib:kyverno_toolchain_type"].kyvernoinfo.bin
    cmd = ""  #"mkdir -p policies && mkdir -p manifests"
    command = [kyverno_bin.short_path, "apply"]
    # for f in ctx.files.manifests:
    #     #parts = command + [f.short_path, "-o", "json", "--file", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/kyverno.json"]
    #     parts = ["cp", f.short_path, "policies/"]
    #     cmd += " ".join([part for part in parts if part]) + "\n"

    # for f in ctx.files.manifests:
    #     #parts = command + [f.short_path, "-o", "json", "--file", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/kyverno.json"]
    #     parts = ["cp", f.short_path, "manifests/"]
    #     cmd += " ".join([part for part in parts if part]) + "\n"
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

# Rule that tests whether a JSON file is valid.
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

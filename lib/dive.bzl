def _dive_impl(ctx):
    dive_bin = ctx.toolchains["@rules_k8s_cd//lib:dive_toolchain_type"].diveinfo.bin
    cmd = ""
    command = [dive_bin.short_path]
    for f in ctx.files.srcs:
        parts = command + [f.short_path, "--source", "docker-archive"]
        cmd += " ".join(["echo", "Load in Docker: docker import", "$BUILD_WORKSPACE_DIRECTORY/%s" % f.path]) + "\n"
        cmd += " ".join([part for part in parts if part]) + "\n"

    for f in ctx.attr.images:
        parts = command + [f]
        cmd += " ".join([part for part in parts if part]) + "\n"

    # Write the file that will be executed by 'bazel test'.
    ctx.actions.write(
        output = ctx.outputs.test,
        content = cmd,
    )

    return [DefaultInfo(
        executable = ctx.outputs.test,
        runfiles = ctx.runfiles(files = [
            dive_bin,
        ] + ctx.files.srcs + ctx.files.manifests),
    )]

# Rule that tests whether a JSON file is valid.
dive = rule(
    implementation = _dive_impl,
    attrs = {
        "srcs": attr.label_list(
            mandatory = False,
            allow_files = [".tar"],
            doc = ("List of inputs. The test will scan all images passed as srcs."),
        ),
        "images": attr.string_list(
            mandatory = False,
            doc = ("List of images. The test will scan all images passed as srcs."),
        ),
        "manifests": attr.label_list(
            mandatory = False,
            allow_files = [".yaml"],
            doc = ("List of manifests. The test will scan all images defined inside manifests."),
        ),
    },
    toolchains = ["@rules_k8s_cd//lib:dive_toolchain_type"],
    outputs = {"test": "%{name}.sh"},
    test = False,
    executable = True,
)

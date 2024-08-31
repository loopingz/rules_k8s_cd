load("//starlark:utils.bzl", "download_binary")

_binaries = {
    "darwin_amd64": ("https://github.com/wagoodman/dive/releases/download/v0.12.0/dive_0.12.0_darwin_amd64.tar.gz", "2f7d0a7f970e09618b87f286c6ccae6a7423331372c6ced15760a5c9d6f27704"),
    "darwin_arm64": ("https://github.com/wagoodman/dive/releases/download/v0.12.0/dive_0.12.0_darwin_arm64.tar.gz", "8ead7ce468f230ffce45b679dd1421945d6e4276654b0d90d389e357af2f4151"),
    "linux_amd64": ("https://github.com/wagoodman/dive/releases/download/v0.12.0/dive_0.12.0_linux_amd64.tar.gz", "20a7966523a0905f950c4fbf26471734420d6788cfffcd4a8c4bc972fded3e96"),
    "linux_arm64": ("https://github.com/wagoodman/dive/releases/download/v0.12.0/dive_0.12.0_linux_arm64.tar.gz", "a2a1470302cdfa367a48f80b67bbf11c0cd8039af9211e39515bd2bbbda58fea"),
}

def dive_setup(name = "dive_bin", binaries = _binaries, bin = ""):
    if (bin == ""):
        bin = name.replace("_bin", "")
    download_binary(name = name, binaries = binaries, bin = bin)

def _dive_impl(ctx):
    cmd = ""
    command = [ctx.executable._dive.short_path]
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
            ctx.executable._dive,
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
        "_dive": attr.label(
            cfg = "host",
            executable = True,
            default = Label("@dive_bin//:dive"),
        ),
    },
    outputs = {"test": "%{name}.sh"},
    test = False,
    executable = True,
)

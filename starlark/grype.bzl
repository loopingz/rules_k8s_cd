load("//starlark:utils.bzl", "download_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.75.0/grype_0.75.0_darwin_amd64.tar.gz", "75c5209610de5b1d1571665707be907d97575ed720622f91b6066596bfac02d4"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.75.0/grype_0.75.0_darwin_arm64.tar.gz", "a61f3994dd20431b18c8bee03d39f17ed3e0989b57426ccb94d57e943c5be0db"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.75.0/grype_0.75.0_linux_amd64.tar.gz", "2a087c9ca09969d0e4aa7a14132b5dd2cdb8f050446c1d4b9095044309ee3aec"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.75.0/grype_0.75.0_linux_arm64.tar.gz", "70b4c7a5153093cbacb4b61f12a2b781b102bb3b99681eb96c4d1d9edded4b07"),
}

def grype_setup(name = "grype_bin", binaries = _binaries, bin = ""):
    if (bin == ""):
        bin = name.replace("_bin", "")
    download_binary(name = name, binaries = binaries, bin = bin)

def _grype_test_impl(ctx):
    cmd = ""
    command = [ctx.executable._grype.short_path]
    for f in ctx.files.srcs:
        cmd += "mkdir -p $BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "\n"
        parts = command + [f.short_path, "-o", "json", "--file", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/grype.json"]
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
            ctx.executable._grype,
        ] + ctx.files.srcs + ctx.files.manifests),
    )]

# Rule that tests whether a JSON file is valid.
grype_scan = rule(
    implementation = _grype_test_impl,
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
        "_grype": attr.label(
            cfg = "host",
            executable = True,
            default = Label("@grype_bin//:grype"),
        ),
    },
    outputs = {"test": "%{name}.sh"},
    test = False,
    executable = True,
)

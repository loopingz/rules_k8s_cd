load("//starlark:utils.bzl", "download_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.71.0/grype_0.70.0_darwin_amd64.tar.gz", "4e04624cb554a47ae33877e110f89aa000679d030154a62baa2f5591a14270e7"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.71.0/grype_0.70.0_darwin_arm64.tar.gz", "728d84835cfe01f556a37e14ac059ed4b997a99c8ee8d39b6c74dd2bf5a960c1"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.71.0/grype_0.70.0_linux_amd64.tar.gz", "663cf80e85a9bfd1056626bd9f69f06ff811dec9281f1b0bb3edc9c48cf34fb4"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.71.0/grype_0.70.0_linux_arm64.tar.gz", "afa8748863397d177a9ff0fb2100a37940aba1776ae6e35916e48bd1ed896837"),
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

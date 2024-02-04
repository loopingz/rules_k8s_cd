load("//starlark:utils.bzl", "download_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.74.4/grype_0.74.4_darwin_amd64.tar.gz", "a287961784efa48b07f499d06671b9b41d1fe70c56341e69cdf168f7c8ca3033"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.74.4/grype_0.74.4_darwin_arm64.tar.gz", "0a7e775ac742e234f4317e80a868bd2fd927fa0b66d617c868b5c71b38e9b416"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.74.4/grype_0.74.4_linux_amd64.tar.gz", "ec180f410d0edb8837f5735acc0d347122a4b235cf9f7ff692b096fc670fd558"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.74.4/grype_0.74.4_linux_arm64.tar.gz", "29faad1775dae7901d5d3cad449f858ca9777c0aff7c39e44c345d0f637ed740"),
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

load("//starlark:utils.bzl", "download_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256

#+ updater:github https://github.com/anchore/grype/releases/latest 
_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.69.1/grype_0.69.1_darwin_amd64.tar.gz", "da6846fe8d722c852e3e0583f0511443219829195f972e84550ef2843b760110"),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.69.1/grype_0.69.1_darwin_arm64.tar.gz", "5b482c0f9806ddda60d8c586952dff9651500bd5a298d1e84b624d7cf25908e3"),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.69.1/grype_0.69.1_linux_amd64.tar.gz", "53ad7a96d0561fdacc82519972025ba0f1a74e16edfdf628d4ce16cc3c714817"),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.69.1/grype_0.69.1_linux_arm64.tar.gz", "22d4fdf249fa9f73d4eff8bb4fb19649171da4cf1eb420422fae1da57065a739"),
}
#- updater:github

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

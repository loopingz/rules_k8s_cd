load("//starlark:utils.bzl", "download_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.74.1/grype_0.74.1_darwin_amd64.tar.gz", "3a311226fba7f6e6974060213588ae079a1d282cfb3fb8c780f40942adbbff35""),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.74.1/grype_0.74.1_darwin_arm64.tar.gz", "4ac9d18dccd837f6a4c7a36cdf0b9c27c7d31cc094f58a71a4781e34919e94e9""),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.74.1/grype_0.74.1_linux_amd64.tar.gz", "03b3728491b4237ac540a9e9b4cfe230efe2a052faa23547b93911614cb5b99c""),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.74.1/grype_0.74.1_linux_arm64.tar.gz", "d7112e2fb5021824086c6f6022f4d3a431a9adf4436a46c1d5ad6bc6a7bd2097""),
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

load("//starlark:utils.bzl", "download_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256

_binaries = {
    "darwin_amd64": ("https://github.com/anchore/grype/releases/download/v0.73.4/grype_0.73.4_darwin_amd64.tar.gz", "9d3251c8c8314f5510df5108204d0275680600d481cef8d480d36d0181d2b798"""),
    "darwin_arm64": ("https://github.com/anchore/grype/releases/download/v0.73.4/grype_0.73.4_darwin_arm64.tar.gz", "cf0a095eca785645b874debc343889aa776a7149cead6dc1da0b41d43c49625a"""),
    "linux_amd64": ("https://github.com/anchore/grype/releases/download/v0.73.4/grype_0.73.4_linux_amd64.tar.gz", "5b18be6d9313552ec2adeef9e7141f1c85a0d3bd15f3bc76bb9846a5bd55116d"""),
    "linux_arm64": ("https://github.com/anchore/grype/releases/download/v0.73.4/grype_0.73.4_linux_arm64.tar.gz", "ea10c210882790509b19f70f9ec821ea8f4c6fafadcc0a0a156a6606d13bff37"""),
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

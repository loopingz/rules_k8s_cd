def _trivy_impl(ctx):
    trivy_bin = ctx.toolchains["@rules_k8s_cd//lib:trivy_toolchain_type"].trivyinfo.bin
    cmd = ""
    command = [trivy_bin.short_path]
    for f in ctx.files.srcs:
        cmd += "mkdir -p $BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "\n"
        parts = command + ["image", "--input", "./" + f.short_path, "-f", "json", "--output", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/trivy.json"]
        cmd += " ".join([part for part in parts if part]) + "\n"
        parts = command + ["image", "--input", "./" + f.short_path, "-f", "table"]
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
            trivy_bin,
        ] + ctx.files.srcs + ctx.files.manifests),
    )]

# Rule that tests whether a JSON file is valid.
trivy_scan = rule(
    implementation = _trivy_impl,
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
    toolchains = ["@rules_k8s_cd//lib:trivy_toolchain_type"],
    outputs = {"test": "%{name}.sh"},
    test = False,
    executable = True,
)

def _trivy_sbom_impl(ctx):
    trivy_bin = ctx.toolchains["@rules_k8s_cd//lib:trivy_toolchain_type"].trivyinfo.bin
    cmd = ""
    command = [trivy_bin.short_path]
    for f in ctx.files.srcs:
        cmd += "mkdir -p $BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "\n"
        parts = command + ["image", "--input", "./" + f.short_path, "--format", "spdx-json", "--output", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/sbom-spdx.json"]
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
            trivy_bin,
        ] + ctx.files.srcs + ctx.files.manifests),
    )]

# Rule that tests whether a JSON file is valid.
trivy_sbom = rule(
    implementation = _trivy_sbom_impl,
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
    toolchains = ["@rules_k8s_cd//lib:trivy_toolchain_type"],
    outputs = {"test": "%{name}.sh"},
    test = False,
    executable = True,
)

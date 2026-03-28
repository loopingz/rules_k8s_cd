"""Factory for creating scanner rules with shared boilerplate."""

def create_scanner_rule(toolchain_type, info_field, build_commands, doc = ""):
    """Creates a scanner rule with standard attrs, outputs, and runfiles.

    Args:
        toolchain_type: The toolchain type label string.
        info_field: The field name on the toolchain to access the tool info (e.g. "grypeinfo").
        build_commands: A callback function(bin_short_path, ctx) that returns the shell command string.
        doc: Optional documentation string for the rule.

    Returns:
        A Bazel rule.
    """

    def _impl(ctx):
        tool_bin = getattr(ctx.toolchains[toolchain_type], info_field).bin
        cmd = build_commands(tool_bin.short_path, ctx)

        # Write the file that will be executed by 'bazel test'.
        ctx.actions.write(
            output = ctx.outputs.test,
            content = cmd,
        )

        return [DefaultInfo(
            executable = ctx.outputs.test,
            runfiles = ctx.runfiles(files = [
                tool_bin,
            ] + ctx.files.srcs + ctx.files.manifests),
        )]

    return rule(
        implementation = _impl,
        doc = doc,
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
        toolchains = [toolchain_type],
        outputs = {"test": "%{name}.sh"},
        test = False,
        executable = True,
    )

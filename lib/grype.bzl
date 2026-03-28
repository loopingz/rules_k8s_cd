load("//lib/private:scanner_factory.bzl", "create_scanner_rule")

def _grype_commands(bin_short_path, ctx):
    cmd = ""
    command = [bin_short_path]
    for f in ctx.files.srcs:
        cmd += "mkdir -p $BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "\n"
        parts = command + [f.short_path, "-o", "json", "--file", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/grype.json"]
        cmd += " ".join([part for part in parts if part]) + "\n"

    for f in ctx.attr.images:
        parts = command + [f]
        cmd += " ".join([part for part in parts if part]) + "\n"

    return cmd

# Rule that scans container images for vulnerabilities using Grype.
grype_scan = create_scanner_rule(
    toolchain_type = "@rules_k8s_cd//lib:grype_toolchain_type",
    info_field = "grypeinfo",
    build_commands = _grype_commands,
    doc = "Rule that scans container images for vulnerabilities using Grype.",
)

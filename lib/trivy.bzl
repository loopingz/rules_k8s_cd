load("//lib/private:scanner_factory.bzl", "create_scanner_rule")

def _trivy_scan_commands(bin_short_path, ctx):
    cmd = ""
    command = [bin_short_path]
    for f in ctx.files.srcs:
        cmd += "mkdir -p $BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "\n"
        parts = command + ["image", "--input", "./" + f.short_path, "-f", "json", "--output", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/trivy.json"]
        cmd += " ".join([part for part in parts if part]) + "\n"
        parts = command + ["image", "--input", "./" + f.short_path, "-f", "table"]
        cmd += " ".join([part for part in parts if part]) + "\n"

    for f in ctx.attr.images:
        parts = command + [f]
        cmd += " ".join([part for part in parts if part]) + "\n"

    return cmd

def _trivy_sbom_commands(bin_short_path, ctx):
    cmd = ""
    command = [bin_short_path]
    for f in ctx.files.srcs:
        cmd += "mkdir -p $BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "\n"
        parts = command + ["image", "--input", "./" + f.short_path, "--format", "spdx-json", "--output", "$BUILD_WORKSPACE_DIRECTORY/security/reports/" + f.short_path.replace("../", "") + "/sbom-spdx.json"]
        cmd += " ".join([part for part in parts if part]) + "\n"

    for f in ctx.attr.images:
        parts = command + [f]
        cmd += " ".join([part for part in parts if part]) + "\n"

    return cmd

# Rule that scans container images for vulnerabilities using Trivy.
trivy_scan = create_scanner_rule(
    toolchain_type = "@rules_k8s_cd//lib:trivy_toolchain_type",
    info_field = "trivyinfo",
    build_commands = _trivy_scan_commands,
    doc = "Rule that scans container images for vulnerabilities using Trivy.",
)

# Rule that generates an SBOM (Software Bill of Materials) using Trivy.
trivy_sbom = create_scanner_rule(
    toolchain_type = "@rules_k8s_cd//lib:trivy_toolchain_type",
    info_field = "trivyinfo",
    build_commands = _trivy_sbom_commands,
    doc = "Rule that generates an SBOM (Software Bill of Materials) using Trivy.",
)

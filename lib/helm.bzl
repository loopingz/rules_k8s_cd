"""Render local Helm charts as Kubernetes YAML, optionally excluding resources."""

def _helm_template_impl(ctx):
    chart = ctx.file.chart
    if chart.basename == "Chart.yaml":
        chart_path = chart.dirname
    elif chart.basename.endswith(".tgz") or chart.basename.endswith(".tar.gz") or chart.is_directory:
        chart_path = chart.path
    else:
        fail("chart must be a Chart.yaml file, a packaged chart (.tgz/.tar.gz), or a directory artifact")

    out = ctx.outputs.out or ctx.actions.declare_file(ctx.label.name + ".yaml")
    args = ctx.actions.args()
    args.add("--helm", ctx.toolchains["@rules_k8s_cd//lib:helm_toolchain_type"].helminfo.bin)
    args.add("--output", out)
    for kind in ctx.attr.exclude_kinds:
        args.add("--exclude-kind", kind)
    for resource in ctx.attr.exclude_resources:
        args.add("--exclude-resource", resource)
    args.add("--")
    args.add("template")
    args.add(ctx.attr.release_name or ctx.label.name)
    args.add(chart_path)
    if ctx.attr.namespace:
        args.add("--namespace", ctx.attr.namespace)
    for value in ctx.files.values:
        args.add("--values", value)
    for key, value in sorted(ctx.attr.set.items()):
        args.add("--set", key + "=" + value)
    for key, value in sorted(ctx.attr.set_string.items()):
        args.add("--set-string", key + "=" + value)
    if ctx.attr.kube_version:
        args.add("--kube-version", ctx.attr.kube_version)
    for version in ctx.attr.api_versions:
        args.add("--api-versions", version)
    if ctx.attr.include_crds:
        args.add("--include-crds")
    if ctx.attr.skip_tests:
        args.add("--skip-tests")

    ctx.actions.run(
        executable = ctx.executable._renderer,
        arguments = [args],
        inputs = depset([chart] + ctx.files.data + ctx.files.values),
        tools = [ctx.toolchains["@rules_k8s_cd//lib:helm_toolchain_type"].helminfo.bin],
        outputs = [out],
        mnemonic = "HelmTemplate",
        progress_message = "Rendering Helm chart %{label}",
    )
    return [DefaultInfo(files = depset([out]))]

helm_template = rule(
    implementation = _helm_template_impl,
    attrs = {
        "chart": attr.label(mandatory = True, allow_single_file = True, doc = "Chart.yaml, a local chart archive, or a generated directory containing the chart."),
        "data": attr.label_list(allow_files = True, doc = "All chart files, including templates, defaults, and vendored dependencies. Use glob([\"chart/**\"])."),
        "values": attr.label_list(allow_files = True, doc = "Values files, applied in order; later files take precedence."),
        "set": attr.string_dict(doc = "Helm --set overrides; Helm interprets value types and escaping."),
        "set_string": attr.string_dict(doc = "Helm --set-string overrides, preserving string values."),
        "release_name": attr.string(doc = "Release name; defaults to the target name."),
        "namespace": attr.string(doc = "Release namespace. Filters match metadata.namespace as rendered, including when it is omitted."),
        "kube_version": attr.string(doc = "Kubernetes version used for Helm capabilities."),
        "api_versions": attr.string_list(doc = "Additional API versions available in Helm capabilities."),
        "include_crds": attr.bool(default = False, doc = "Include files from the chart's crds directory."),
        "skip_tests": attr.bool(default = False, doc = "Omit Helm test hooks."),
        "exclude_kinds": attr.string_list(doc = "Exact, case sensitive kinds to remove, e.g. [\"Secret\", \"Job\"]."),
        "exclude_resources": attr.string_list(doc = "Case sensitive glob patterns Kind/name (all namespaces) or Kind/namespace/name. Each component supports *, ?, and character classes; an empty namespace matches omitted metadata.namespace."),
        "out": attr.output(doc = "Output YAML; defaults to <name>.yaml."),
        "_renderer": attr.label(default = Label("//go/helm_template:helm_template"), cfg = "exec", executable = True),
    },
    toolchains = ["@rules_k8s_cd//lib:helm_toolchain_type"],
    doc = "Render a local Helm chart at build time and remove selected resources. Dependencies must be vendored; rendering does not fetch charts or use local Helm configuration.",
)

"""Helm support for rules_k8s_cd.

Provides:
- helm_pull: module extension to pull Helm charts from classic HTTP Helm repos
- helm_template: rule that renders charts, filters, and patches (added in a later task)
"""

# ---------- helm_pull (repository rule + module extension) ----------

def _helm_chart_repo_impl(rctx):
    url = rctx.attr.url
    if url == "":
        url = "%s/%s-%s.tgz" % (rctx.attr.repo.rstrip("/"), rctx.attr.chart, rctx.attr.version)

    tarball_name = "%s-%s.tgz" % (rctx.attr.chart, rctx.attr.version)
    res = rctx.download(
        url = url,
        output = tarball_name,
        sha256 = rctx.attr.sha256,
    )
    computed_sha = res.sha256

    if rctx.attr.sha256 == "":
        # buildifier: disable=print
        print(("WARNING: helm_pull.chart(name = \"%s\") has no sha256 pinned.\n" +
               "         Computed: sha256 = \"%s\"\n" +
               "         Add this to MODULE.bazel for reproducibility.") % (rctx.name, computed_sha))

    rctx.extract(archive = tarball_name)

    rctx.file("sha256.txt", "sha256 = \"%s\"\n" % computed_sha)

    # Embed the sha256 value directly so the script needs no runfiles lookup
    rctx.file("print_sha.sh", "#!/usr/bin/env bash\nprintf 'sha256 = \"%s\"\\n'\n" % computed_sha, executable = True)

    rctx.file("BUILD.bazel", """load("@rules_shell//shell:sh_binary.bzl", "sh_binary")

package(default_visibility = ["//visibility:public"])

filegroup(
    name = "chart",
    srcs = glob(["{chart}/**/*"], allow_empty = False),
)

exports_files(["sha256.txt"])

sh_binary(
    name = "sha256",
    srcs = ["print_sha.sh"],
)
""".format(chart = rctx.attr.chart))

helm_chart_repo = repository_rule(
    implementation = _helm_chart_repo_impl,
    attrs = {
        "repo":    attr.string(mandatory = True,  doc = "Base URL of the Helm repo"),
        "chart":   attr.string(mandatory = True,  doc = "Chart name"),
        "version": attr.string(mandatory = True,  doc = "Chart version"),
        "sha256":  attr.string(default = "",      doc = "Expected sha256 of the tarball (optional; a warning is emitted if absent)"),
        "url":     attr.string(default = "",      doc = "Optional explicit tarball URL (defaults to <repo>/<chart>-<version>.tgz)"),
    },
)

def _helm_pull_ext_impl(mctx):
    for mod in mctx.modules:
        for tag in mod.tags.chart:
            helm_chart_repo(
                name    = tag.name,
                repo    = tag.repo,
                chart   = tag.chart,
                version = tag.version,
                sha256  = tag.sha256,
                url     = tag.url,
            )

helm_pull = module_extension(
    implementation = _helm_pull_ext_impl,
    tag_classes = {
        "chart": tag_class(attrs = {
            "name":    attr.string(mandatory = True),
            "repo":    attr.string(mandatory = True),
            "chart":   attr.string(mandatory = True),
            "version": attr.string(mandatory = True),
            "sha256":  attr.string(default = ""),
            "url":     attr.string(default = ""),
        }),
    },
)

# ---------- helm_template rule ----------

# HelmInfo provider is exposed by the toolchain defined in lib/private/helm_toolchain.bzl.
# The helm binary is available via the toolchain at `@rules_k8s_cd//lib:helm_toolchain_type`.

def _helm_template_impl(ctx):
    chart_files = ctx.files.chart
    if not chart_files:
        fail("helm_template: `chart` has no files")

    # The chart directory is the parent of Chart.yaml. Accept either a
    # filegroup covering all chart files or a single directory label.
    chart_root = None
    for f in chart_files:
        if f.basename == "Chart.yaml":
            chart_root = f.dirname
            break
    if chart_root == None:
        chart_root = chart_files[0].dirname

    rendered = ctx.actions.declare_file(ctx.label.name + "_rendered.yaml")

    helm_toolchain = ctx.toolchains["@rules_k8s_cd//lib:helm_toolchain_type"]
    helm_bin = helm_toolchain.helminfo.helm

    release = ctx.attr.release_name if ctx.attr.release_name else ctx.label.name

    # Action 1: helm template -> rendered.yaml (via shell wrapper for stdout redirect).
    helm_cmd_args = [helm_bin.path, "template", release, chart_root, "--namespace", ctx.attr.namespace]
    for v in ctx.files.values:
        helm_cmd_args += ["-f", v.path]

    ctx.actions.run_shell(
        outputs = [rendered],
        inputs = chart_files + ctx.files.values,
        tools = [helm_bin],
        command = '"$@" > "%s"' % rendered.path,
        arguments = helm_cmd_args,
        mnemonic = "HelmTemplate",
        progress_message = "helm template %s" % ctx.label,
    )

    # Action 2: helm_postrender -> final output.
    out = ctx.outputs.out
    postrender = ctx.executable._postrender

    pr_args = ctx.actions.args()
    pr_args.add("--in", rendered.path)
    pr_args.add("--out", out.path)
    for s in ctx.attr.exclude:
        pr_args.add("--exclude", s)
    for p in ctx.files.patchesStrategicMerge:
        pr_args.add("--patch-sm", p.path)
    for p in ctx.files.patchesJson6902:
        pr_args.add("--patch-json6902", p.path)

    ctx.actions.run(
        executable = postrender,
        arguments = [pr_args],
        inputs = [rendered] + ctx.files.patchesStrategicMerge + ctx.files.patchesJson6902,
        outputs = [out],
        mnemonic = "HelmPostRender",
        progress_message = "helm_postrender %s" % ctx.label,
    )

    return [DefaultInfo(files = depset([out]))]

_helm_template = rule(
    implementation = _helm_template_impl,
    attrs = {
        "chart": attr.label(mandatory = True, allow_files = True, doc = "Chart directory (typically `@name//:chart` from helm_pull) or a filegroup covering a local chart"),
        "values": attr.label_list(allow_files = True, doc = "Values YAML files, merged in order (later overrides earlier)"),
        "release_name": attr.string(default = "", doc = "Helm release name; defaults to target name"),
        "namespace": attr.string(default = "default", doc = "Namespace for rendering"),
        "exclude": attr.string_list(doc = "Pre-serialized exclude selectors (set by the helm_template macro)"),
        "patchesStrategicMerge": attr.label_list(allow_files = True, doc = "Strategic-merge patch files"),
        "patchesJson6902": attr.label_list(allow_files = True, doc = "JSON6902 patch files (target + patch format)"),
        "out": attr.output(mandatory = True, doc = "Output YAML file path"),
        "_postrender": attr.label(
            default = Label("//go/helm_postrender:helm_postrender"),
            cfg = "exec",
            executable = True,
        ),
    },
    toolchains = ["@rules_k8s_cd//lib:helm_toolchain_type"],
)

def helm_template(name, exclude = None, out = None, **kwargs):
    """Render a Helm chart, then optionally filter and patch the output.

    Args:
      name: Target name.
      exclude: List of selector dicts (each with optional keys apiVersion/kind/name/namespace).
               Resources matching ANY selector are dropped. Empty/None disables filtering.
      out: Output YAML file name (defaults to <name>.yaml).
      **kwargs: Passed through: chart, values, release_name, namespace,
                patchesStrategicMerge, patchesJson6902.
    """
    serialized = []
    if exclude:
        for sel in exclude:
            parts = []
            for k in ("apiVersion", "kind", "name", "namespace"):
                if k in sel:
                    parts.append("%s=%s" % (k, sel[k]))
            if not parts:
                fail("helm_template: exclude dict must contain at least one of apiVersion/kind/name/namespace")
            serialized.append(",".join(parts))

    _helm_template(
        name = name,
        exclude = serialized,
        out = out if out else (name + ".yaml"),
        **kwargs
    )

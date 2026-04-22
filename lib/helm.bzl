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

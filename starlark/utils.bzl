load(
    "@io_bazel_rules_docker//skylib:path.bzl",
    _get_runfile_path = "runfile",
)
load("@aspect_bazel_lib//lib:repo_utils.bzl", "repo_utils")

def _download_binary_impl(ctx):
    platform = repo_utils.platform(ctx)
    if platform not in ctx.attr.binaries:
        fail("Platform " + platform + " is not supported")
    path = ctx.path("bin")

    url, sha256 = ctx.attr.binaries[platform]

    if (url.endswith(".tar.gz")):
        ctx.file("BUILD", """
sh_binary(
    name = "{}",
    srcs = ["download/{}"],
    visibility = ["//visibility:public"],
)
""".format(ctx.attr.bin, ctx.attr.bin))      
        ctx.download_and_extract(url, "download/", sha256 = sha256)
    else:
        ctx.file("BUILD", """
sh_binary(
    name = "{}",
    srcs = ["{}"],
    visibility = ["//visibility:public"],
)
""".format(ctx.attr.name, ctx.attr.bin))      
        ctx.download(url, ctx.attr.bin, sha256 = sha256, executable = True)


download_binary = repository_rule(
    _download_binary_impl,
    attrs = {
        "binaries": attr.string_list_dict(),
        "bin": attr.string(),
    }
)


# Inspired by https://github.com/bazelbuild/rules_k8s/blob/master/k8s/objects.bzl
def _runfiles(ctx, f):
    return "PYTHON_RUNFILES=${RUNFILES} ${RUNFILES}/%s $@" % _get_runfile_path(ctx, f)

def _run_all_impl(ctx):
    if ctx.attr.wrap_exits:
        _prefix = "code=0"
        _append = " || code=1"
        _suffix = "exit $code"
    else:
        _prefix = ""
        _append = ""
        _suffix = ""

    if ctx.attr.parralel:
        _prefix = _prefix + " async "

    _statements = ("\n" + ctx.attr.delimiter).join([_prefix] +
                                                   [_runfiles(ctx, exe.files_to_run.executable) + _append for exe in ctx.attr.targets] +
                                                   [_suffix])

    if ctx.attr.parralel:
        _statements += "\nwaitpids\n"

    ctx.actions.expand_template(
        template = ctx.file._template,
        substitutions = {
            "%{resolve_statements}": _statements,
        },
        output = ctx.outputs.executable,
    )

    runfiles = [obj.files_to_run.executable for obj in ctx.attr.targets]
    for obj in ctx.attr.targets:
        runfiles.extend(list(obj.default_runfiles.files.to_list()))

    return [
        DefaultInfo(
            runfiles = ctx.runfiles(files = runfiles),
        ),
    ]

run_all = rule(
    attrs = {
        "delimiter": attr.string(default = ""),
        "targets": attr.label_list(
            cfg = "target",
            allow_empty = False,
        ),
        "wrap_exits": attr.bool(default = False),
        "parralel": attr.bool(default = True),
        "_template": attr.label(
            default = Label(":resolve-all.sh.tpl"),
            allow_single_file = True,
        ),
    },
    executable = True,
    implementation = _run_all_impl,
)

# Show rule
#
# Useful for debug to have a way to display the output of a target

def _show_impl(ctx):
    script_content = "#!/usr/bin/env bash\nset -e\n"

    outputs = [
        "tree -C `dirname %s` -I %s" % (ctx.attr.src.files.to_list()[0].short_path, ctx.outputs.executable.path.split("/")[-1])
    ]
    if ctx.attr.content:
        for dep in ctx.attr.src.files.to_list():
            outputs.append("echo ---- %s -----\ncat %s\necho \n" % (dep.short_path,dep.short_path))

    script_content += "\n".join(outputs)

    ctx.actions.write(ctx.outputs.executable, script_content, is_executable = True)
    return [
        DefaultInfo(executable = ctx.outputs.executable, runfiles=ctx.runfiles(ctx.attr.src.files.to_list())),
    ]

show = rule(
    implementation = _show_impl,
    attrs = {
        "src": attr.label(
            doc = "Input file(s).",
            mandatory = True,
        ),
        "content": attr.bool(
            default = False
        )
    },
    executable = True,
)
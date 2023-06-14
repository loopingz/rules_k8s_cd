load(
    "@io_bazel_rules_docker//skylib:path.bzl",
    _get_runfile_path = "runfile",
)

def _download_binary_impl(ctx):
    arch = "arm64" if ctx.os.arch == "aarch64" else ctx.os.arch
    osname = "darwin" if ctx.os.name == "mac os x" else ctx.os.name
    platform = osname + "-" + arch
    if platform not in ctx.attr.binaries:
        fail("Platform " + platform + " is not supported")
    path = ctx.path("bin")

    url, sha256 = ctx.attr.binaries[platform]

    if (url.endswith(".tar.gz")):
        ctx.file("BUILD", """
sh_binary(
    name = "{}",
    srcs = ["download/{}/{}"],
    visibility = ["//visibility:public"],
)
""".format(ctx.attr.name, platform, ctx.attr.bin))      
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
                                                   [_runfiles(ctx, exe.files_to_run.executable) + _append for exe in ctx.attr.runners] +
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

    runfiles = [obj.files_to_run.executable for obj in ctx.attr.runners]
    for obj in ctx.attr.runners:
        runfiles.extend(list(obj.default_runfiles.files.to_list()))

    return [
        DefaultInfo(
            runfiles = ctx.runfiles(files = runfiles),
        ),
    ]

run_all = rule(
    attrs = {
        "delimiter": attr.string(default = ""),
        "runners": attr.label_list(
            cfg = "target",
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
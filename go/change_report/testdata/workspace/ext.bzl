"""Root-module extension whose generated repo depends on //:images.txt."""

def _myrepo_impl(rctx):
    rctx.file("images.txt", rctx.attr.content)
    rctx.file("BUILD.bazel", 'filegroup(name = "files", srcs = ["images.txt"], visibility = ["//visibility:public"])\n')

_myrepo = repository_rule(
    implementation = _myrepo_impl,
    attrs = {"content": attr.string()},
)

def _myext_impl(module_ctx):
    # Reading a workspace file records it as FILE:@@//images.txt in the
    # lockfile's recordedInputs.
    _myrepo(name = "myrepo", content = module_ctx.read(Label("//:images.txt")))

myext = module_extension(implementation = _myext_impl)

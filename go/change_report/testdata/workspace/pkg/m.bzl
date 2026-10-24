"""Macro whose glob makes pkg/manifests part of //pkg:viamacro."""

def manifests(name):
    native.filegroup(
        name = name,
        srcs = native.glob(["manifests/*.yaml"]),
        visibility = ["//visibility:public"],
    )

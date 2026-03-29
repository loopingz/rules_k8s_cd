package(default_visibility = ["//visibility:public"])

filegroup(
    name = "baseline",
    srcs = glob(["baseline/*.yaml"]),
)

# Restricted is a superset of baseline per Kubernetes Pod Security Standards.
filegroup(
    name = "restricted",
    srcs = glob([
        "baseline/*.yaml",
        "restricted/*.yaml",
    ]),
)

filegroup(
    name = "best-practices",
    srcs = glob(["best-practices/*.yaml"]),
)

load("@com_github_bazelbuild_buildtools//buildifier:def.bzl", "buildifier")
load("@bazel_skylib//rules:common_settings.bzl", "string_flag")
load("@com_adobe_rules_gitops//skylib:stamp.bzl", "more_stable_status")

string_flag(
    name = "environment",
    build_setting_default = "dev",
)

config_setting(
    name = "dev_flag",
    flag_values = {":environment": "dev"},
)

config_setting(
    name = "beta_flag",
    flag_values = {":environment": "beta"},
)

config_setting(
    name = "preview_flag",
    flag_values = {":environment": "preview"},
)

config_setting(
    name = "prod_flag",
    flag_values = {":environment": "prod"},
)

buildifier(
    name = "buildifier",
)


environments = [
    "dev", 
    "beta",
]

more_stable_status(
    name = "more_stable_status",
    vars = [
        "BUILD_USER",
        "STABLE_GIT_COMMIT",
    ],
    visibility = ["//visibility:public"],
)

# Copyright 2020 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

load("@rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "loopingz.com/rules_k8s_cd/release",
    visibility = ["//visibility:private"],
    deps = [
        "//go/change_report/pkg/bazelq",
        "//go/change_report/pkg/cache",
        "//go/change_report/pkg/cliutil",
        "//go/change_report/pkg/gitx",
        "//go/change_report/pkg/report",
        "//go/release/pkg/attrib",
        "//go/release/pkg/component",
        "//go/release/pkg/forge",
        "//go/release/pkg/release",
    ],
)

go_binary(
    name = "release",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "release_test",
    srcs = ["main_test.go"],
    embed = [":go_default_library"],
    deps = [
        "//go/change_report/pkg/cache",
        "//go/change_report/pkg/gitx",
        "//go/release/pkg/attrib",
        "//go/release/pkg/component",
        "//go/release/pkg/release",
        "//go/release/pkg/semver",
    ],
)

# End-to-end test against a real Bazel, git and a fake GitHub API. Run it with
#   cd go && go test -tags integration ./release/ -run Integration -v -timeout 30m
# It is manual and not expected to pass under `bazel test` (Bazel inside the
# Bazel sandbox is not supported).
go_test(
    name = "integration_test",
    srcs = ["integration_test.go"],
    data = glob(["testdata/**"]),
    embed = [":go_default_library"],
    gotags = ["integration"],
    tags = ["manual"],
    deps = ["//go/release/pkg/release"],
)

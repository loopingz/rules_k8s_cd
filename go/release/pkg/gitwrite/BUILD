# Copyright 2020 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "gitwrite",
    srcs = ["gitwrite.go"],
    importpath = "loopingz.com/rules_k8s_cd/release/pkg/gitwrite",
    visibility = ["//go/release:__subpackages__"],
    deps = ["//go/change_report/pkg/gitx"],
)

go_test(
    name = "gitwrite_test",
    srcs = ["gitwrite_test.go"],
    embed = [":gitwrite"],
    deps = ["//go/change_report/pkg/gitx"],
)

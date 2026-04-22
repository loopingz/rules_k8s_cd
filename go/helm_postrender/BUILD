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
    srcs = [
        "doc.go",
        "filter.go",
        "main.go",
        "patch.go",
    ],
    importpath = "loopingz.com/rules_k8s_cd/helm_postrender",
    visibility = ["//visibility:private"],
    deps = [
        "//go/vendor/github.com/evanphx/json-patch/v5:json-patch",
        "//go/vendor/k8s.io/api/core/v1:v1",
        "//go/vendor/k8s.io/apimachinery/pkg/util/strategicpatch:strategicpatch",
        "//go/vendor/sigs.k8s.io/yaml:yaml",
    ],
)

go_binary(
    name = "helm_postrender",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "helm_postrender_test",
    srcs = [
        "doc_test.go",
        "filter_test.go",
        "patch_test.go",
    ],
    embed = [":go_default_library"],
)

load("@bazel_skylib//rules:diff_test.bzl", "diff_test")
load("//lib:release.bzl", "release_component", "release_version", "release_version_file")

filegroup(
    name = "demo_files",
    srcs = ["data.txt"],
)

# Released component with a VERSION file (1.2.3).
release_component(
    name = "demo",
    targets = [":demo_files"],
)

# Never-released component: its VERSION file does not exist, which must not
# break the build. Also exercises targets_query, bump_files, release_targets.
release_component(
    name = "fresh",
    bump_files = {"data.txt": "^(hello)$"},
    changelog = "FRESH_CHANGELOG.md",
    ignore = ["data.txt"],
    release_targets = [":demo_files"],
    tag_format = "fresh-v{version}",
    targets_query = "//lib/tests/release:demo_files",
    untagged_version = "dev-{component}",
    version_file = "FRESH_VERSION",
)

# `bazel test` inherits `build --stamp` from .bazelrc: the expectation for
# :demo.version is checked on an unstamped twin so it does not depend on git.
release_version(
    name = "demo_unstamped.version",
    component = "demo",
    stamp = 0,
    untagged_version = "main-{commit}",
    version_file = "VERSION",
)

diff_test(
    name = "demo_version_test",
    file1 = ":demo_unstamped.version",
    file2 = "expected/demo.version.txt",
)

# Untagged template without git variables: deterministic even when stamped.
diff_test(
    name = "fresh_version_test",
    file1 = ":fresh.version",
    file2 = "expected/fresh.version.txt",
)

release_version_file(
    name = "version_go",
    out = "version.go",
    template = "//lib/release:version.go.tpl",
    version = ":demo_unstamped.version",
)

diff_test(
    name = "version_go_test",
    file1 = ":version.go",
    file2 = "expected/version.go",
)

release_version_file(
    name = "version_main_go",
    out = "version_main.go",
    substitutions = {"package": "main"},
    template = "//lib/release:version.go.tpl",
    version = ":demo_unstamped.version",
)

diff_test(
    name = "version_main_go_test",
    file1 = ":version_main.go",
    file2 = "expected/version_main.go",
)

release_version_file(
    name = "version_ts",
    out = "version.ts",
    template = "//lib/release:version.ts.tpl",
    version = ":demo_unstamped.version",
)

diff_test(
    name = "version_ts_test",
    file1 = ":version.ts",
    file2 = "expected/version.ts",
)

release_version_file(
    name = "version_py",
    out = "version.py",
    template = "//lib/release:version.py.tpl",
    version = ":demo_unstamped.version",
)

diff_test(
    name = "version_py_test",
    file1 = ":version.py",
    file2 = "expected/version.py",
)

release_version_file(
    name = "version_json",
    out = "version.json",
    template = "//lib/release:version.json.tpl",
    version = ":demo_unstamped.version",
)

diff_test(
    name = "version_json_test",
    file1 = ":version.json",
    file2 = "expected/version.json",
)

# The tool discovers components with `kind("_release_component rule", ...)`.
genquery(
    name = "release_component_query",
    expression = "kind('_release_component rule', deps(//lib/tests/release:demo))",
    opts = ["--output=label_kind"],
    scope = [":demo"],
)

diff_test(
    name = "release_component_query_test",
    file1 = ":release_component_query",
    file2 = "expected/query.txt",
)

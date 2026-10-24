"""Release components: declaration, build-time version resolution and version files.

A `release_component` declares a releasable unit of the repository. The
`release` tool (//go/release) discovers components by querying
`kind("_release_component rule", ...)` and reads their attributes; the
`<name>.version` target resolves the component's version at build time from
the Bazel stamp (see lib/stamp.sh):

1. `--embed_label` equal to the component's tag for VERSION -> VERSION;
2. else the component's tag for VERSION at HEAD -> VERSION;
3. else the `untagged_version` template (`main-{commit}` by default).

Unstamped builds (`--nostamp` or `stamp = 0`) resolve every git value to
`unknown`, e.g. `main-unknown`.
"""

load("@aspect_bazel_lib//lib:stamping.bzl", "STAMP_ATTRS", "maybe_stamp")
load("@bazel_skylib//lib:dicts.bzl", "dicts")

ReleaseVersionInfo = provider(
    doc = "Version of a release component resolved at build time.",
    fields = {
        "txt": "File: the version followed by a newline.",
        "json": "File: JSON info (version, commit, commit_full, date, date_iso, branch, component, tagged).",
        "component": "str: the component name.",
    },
)

_STAMPVER = attr.label(
    default = Label("//go/release/stampver"),
    cfg = "exec",
    executable = True,
)

def _release_component_impl(_ctx):
    return [DefaultInfo()]

# Holds a component's configuration for the release tool, which reads it with
# `bazel query`. It produces no outputs.
_release_component = rule(
    implementation = _release_component_impl,
    attrs = {
        "targets": attr.label_list(allow_files = True, doc = "Targets whose changes release the component"),
        "targets_query": attr.string(doc = "Query selecting the targets, instead of targets"),
        "version_file": attr.string(doc = "Workspace-relative path of the VERSION file"),
        "changelog": attr.string(doc = "Workspace-relative path of the changelog"),
        "untagged_version": attr.string(doc = "Version template for untagged builds"),
        "initial_version": attr.string(doc = "Version of the first release"),
        "ignore": attr.string_list(doc = "Paths ignored when computing changes"),
        "tag_format": attr.string(doc = "Tag format, with {name} and {version}"),
        "bump_files": attr.string_dict(doc = "Files to update on release: path -> regex"),
        "release_targets": attr.string_list(doc = "Targets run after tagging (normalised labels)"),
    },
)

def _release_version_impl(ctx):
    txt = ctx.actions.declare_file(ctx.label.name + ".txt")
    js = ctx.actions.declare_file(ctx.label.name + ".json")
    args = ctx.actions.args()
    args.add("version")
    args.add("--component", ctx.attr.component)
    args.add("--tag-format", ctx.attr.tag_format)
    args.add("--untagged", ctx.attr.untagged_version)
    args.add("--out", txt)
    args.add("--out-json", js)
    inputs = []
    if ctx.file.version_file:
        args.add("--version-file", ctx.file.version_file)
        inputs.append(ctx.file.version_file)
    elif ctx.attr.version_path:
        # The file does not exist (never-released component): stampver
        # treats a missing file as "no version".
        args.add("--version-file", ctx.attr.version_path)
    stamp = maybe_stamp(ctx)
    if stamp:
        args.add("--stable-status", stamp.stable_status_file)
        args.add("--volatile-status", stamp.volatile_status_file)
        inputs.extend([stamp.stable_status_file, stamp.volatile_status_file])
    ctx.actions.run(
        executable = ctx.executable._stampver,
        arguments = [args],
        inputs = inputs,
        outputs = [txt, js],
        mnemonic = "ReleaseVersion",
        progress_message = "Resolving version of %s" % ctx.attr.component,
    )
    return [
        DefaultInfo(files = depset([txt])),
        ReleaseVersionInfo(txt = txt, json = js, component = ctx.attr.component),
    ]

# Resolves a component's version at build time. Outputs `<name>.txt` (default
# output) and `<name>.json`, and provides ReleaseVersionInfo.
release_version = rule(
    implementation = _release_version_impl,
    attrs = dicts.add({
        "component": attr.string(mandatory = True, doc = "Component name, as in its tags"),
        "version_file": attr.label(allow_single_file = True, doc = "VERSION file; omit when it does not exist yet"),
        "version_path": attr.string(doc = "Path reported for a missing version_file"),
        "tag_format": attr.string(default = "{name}@v{version}", doc = "Tag format, with {name} and {version}"),
        "untagged_version": attr.string(default = "main-{commit}", doc = "Version template for untagged builds"),
        "_stampver": _STAMPVER,
    }, **STAMP_ATTRS),
)

def _release_version_file_impl(ctx):
    info = ctx.attr.version[ReleaseVersionInfo]
    subs = dicts.add({"package": "version"}, ctx.attr.substitutions)
    args = ctx.actions.args()
    args.add("render")
    args.add("--info", info.json)
    args.add("--template", ctx.file.template)
    args.add("--out", ctx.outputs.out)
    for k in sorted(subs.keys()):
        args.add("--sub", "%s=%s" % (k, subs[k]))
    ctx.actions.run(
        executable = ctx.executable._stampver,
        arguments = [args],
        inputs = [info.json, ctx.file.template],
        outputs = [ctx.outputs.out],
        mnemonic = "ReleaseVersionFile",
        progress_message = "Rendering %{output}",
    )
    return [DefaultInfo(files = depset([ctx.outputs.out]))]

# Renders `template` with the version of `version` ({version}, {commit},
# {commit_full}, {date}, {date_iso}, {branch}, {component}) and the extra
# `substitutions` ({package} defaults to "version"). Templates for Go, TS,
# Python and JSON are in //lib/release.
release_version_file = rule(
    implementation = _release_version_file_impl,
    attrs = {
        "version": attr.label(mandatory = True, providers = [ReleaseVersionInfo], doc = "A release_version target"),
        "template": attr.label(mandatory = True, allow_single_file = True, doc = "Template file"),
        "out": attr.output(mandatory = True, doc = "Output file"),
        "substitutions": attr.string_dict(doc = "Extra {key} substitutions, overriding the version info"),
        "_stampver": _STAMPVER,
    },
)

def _workspace_path(path):
    pkg = native.package_name()
    return pkg + "/" + path if pkg else path

def release_component(
        name,
        targets = [],
        targets_query = "",
        version_file = "VERSION",
        changelog = "CHANGELOG.md",
        untagged_version = "main-{commit}",
        initial_version = "0.1.0",
        ignore = [],
        tag_format = "{name}@v{version}",
        bump_files = {},
        release_targets = [],
        visibility = None):
    """Declares a release component and its `<name>.version` target.

    Args:
      name: component name, used in tags, the release branch and commits.
      targets: targets whose changes release the component (exclusive with targets_query).
      targets_query: Bazel query selecting those targets (exclusive with targets).
      version_file: VERSION file, relative to the package; may not exist yet.
      changelog: changelog file, relative to the package.
      untagged_version: version template for untagged builds ({commit},
        {commit_full}, {branch}, {date}, {component}).
      initial_version: version of the first release.
      ignore: paths ignored when computing changes.
      tag_format: tag format, with {name} and {version}.
      bump_files: files updated on release, path -> regex.
      release_targets: targets run after the component is tagged.
      visibility: visibility of the generated targets.
    """
    if bool(targets) == bool(targets_query):
        fail("release_component %s: exactly one of targets or targets_query must be set" % name)

    _release_component(
        name = name,
        targets = targets,
        targets_query = targets_query,
        version_file = _workspace_path(version_file),
        changelog = _workspace_path(changelog),
        untagged_version = untagged_version,
        initial_version = initial_version,
        ignore = ignore,
        tag_format = tag_format,
        bump_files = bump_files,
        release_targets = [str(native.package_relative_label(t)) for t in release_targets],
        visibility = visibility,
    )

    # A never-released component has no VERSION file yet.
    existing = native.glob([version_file], allow_empty = True)
    release_version(
        name = name + ".version",
        component = name,
        version_file = ":" + version_file if existing else None,
        version_path = _workspace_path(version_file),
        tag_format = tag_format,
        untagged_version = untagged_version,
        visibility = visibility,
    )

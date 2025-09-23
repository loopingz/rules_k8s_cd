load("@rules_oci//oci:extensions.bzl", "oci")  # we'll call oci.pull from our ext
load("@rules_oci//oci:pull.bzl", "oci_pull")
load("@rules_oci//oci/private:push.bzl", "oci_push_lib")

ContainerPushInfo = provider(fields = ["registry", "repository", "digestfile", "tags", "name"])

def image_pushes(images, image_repository, name_suffix = ".push"):
    #image_pushes = dict()

    def process_image(image_label, legacy_name):
        rule_name_parts = [image_repository, legacy_name]
        rule_name = "_".join(rule_name_parts)
        rule_name = rule_name.replace("/", "_").replace(":", "_").replace("@", "_")
        if rule_name.startswith("_"):
            rule_name = rule_name[1:]
        rule_name = "_push." + rule_name
        if not native.existing_rule(rule_name):
            oci_push_info(
                name = rule_name,
                image = image_label,  # buildifier: disable=uninitialized
                repository = image_repository + "/" + legacy_name,
            )
        return ":" + rule_name

    #for image_name in images:
    #    image_pushes[image_name] = process_image(images[image_name], image_name)
    return [process_image(images[image_name], image_name) for image_name in images]

def _oci_push_info(ctx):
    push_result = oci_push_lib.implementation(ctx)

    registry = ctx.attr.repository.split("/")[0]
    repository = "/".join(ctx.attr.repository.split("/")[1:])
    digestfile = ctx.actions.declare_file(ctx.attr.name + ".digest")
    image = ctx.file.image
    yq_bin = ctx.toolchains["@aspect_bazel_lib//lib:yq_toolchain_type"].yqinfo.bin
    executable = ctx.actions.declare_file(ctx.attr.name + ".digest.sh")

    substitutions = {
        "{{yq}}": yq_bin.path,
        "{{image_dir}}": image.path,
        "{{digestfile}}": digestfile.path,
    }

    ctx.actions.expand_template(
        template = ctx.file._oci_push_info_sh,
        output = executable,
        is_executable = True,
        substitutions = substitutions,
    )

    ctx.actions.run(
        executable = executable,
        inputs = [image],
        outputs = [digestfile],
        tools = [yq_bin],
        mnemonic = "OCIPushInfo",
        progress_message = "OCI PushInfo %{label}",
        use_default_shell_env = True,
    )

    return [push_result, ContainerPushInfo(
        registry = registry,
        repository = repository,
        name = repository.split("/").pop(),
        digestfile = digestfile,
        tags = [],
    )]

oci_push_info = rule(
    implementation = _oci_push_info,
    attrs = dict(
        oci_push_lib.attrs,
        _oci_push_info_sh = attr.label(allow_single_file = True, default = "//lib:oci_push_info.sh.tpl"),
    ),
    toolchains = oci_push_lib.toolchains + ["@aspect_bazel_lib//lib:yq_toolchain_type"],
    executable = True,
    outputs = {
        "digest": "%{name}.digest",
    },
)

def _extras_repo_impl(repo_ctx):
    pulled = repo_ctx.attr.pulled_repo  # name of the @repo created by oci.pull
    image_label = "@%s//:%s" % (pulled, pulled)  # common target name; adjust if you prefer

    repo_ctx.file(
        "BUILD.bazel",
        """
load("@rules_oci//oci:defs.bzl", "oci_load")
load("@rules_k8s_cd//lib:dive.bzl", "dive")
load("@rules_k8s_cd//lib:grype.bzl", "grype_scan")
load("@rules_k8s_cd//lib:trivy.bzl", "trivy_scan", "trivy_sbom")

alias(
    name = "base",
    actual = "%s",
    visibility = ["//visibility:public"],
)
alias(
    name = "%s",
    actual = "%s",
    visibility = ["//visibility:public"],
)
# Load the image into your local Docker/CRI
oci_load(
    name = "load",
    image = ":base",
    visibility = ["//visibility:public"],
    repo_tags = []
)

filegroup(
    name = "tarball",
    srcs = [":load"],
    output_group = "tarball",
)

dive(
    name = "dive",
    srcs = [":tarball"],
    visibility = ["//visibility:public"],
)

grype_scan(
    name = "grype",
    srcs = [":tarball"],
    visibility = ["//visibility:public"],
)
trivy_scan(
    name = "trivy",
    srcs = [":tarball"],
    visibility = ["//visibility:public"],
)
trivy_sbom(
    name = "sbom",
    srcs = [":tarball"],
    visibility = ["//visibility:public"],
)
""" % (image_label, repo_ctx.name.split("+")[-1], image_label),
    )

extras_repo = repository_rule(
    implementation = _extras_repo_impl,
    attrs = {
        "pulled_repo": attr.string(mandatory = True),
    },
)

def _oci_plus_impl(ctx):
    created = []
    for mod in ctx.modules:  # ← module_ctx.modules is the right API
        for p in getattr(mod.tags, "pull", []):  # ← no ctx.tags at top-level
            raw_name = p.name + "_original"
            oci_pull(
                name = raw_name,
                image = p.image,
                digest = p.digest,
                tag = p.tag,
                platforms = p.platforms,
            )
            extras_repo(name = p.name, pulled_repo = raw_name)
            created.extend([raw_name, p.name])

    # Advertise to Bazel which repos the root module should import.
    # (Users still write use_repo(...) in MODULE.bazel, or run `bazel mod tidy`.)
    return ctx.extension_metadata(
        root_module_direct_deps = created,
        root_module_direct_dev_deps = [],  # must provide both; keep disjoint
    )

oci_utils = module_extension(
    implementation = _oci_plus_impl,
    tag_classes = {
        # mirror the subset of attrs you need from oci.pull
        "pull": tag_class(
            attrs = {
                "name": attr.string(mandatory = True),
                "image": attr.string(),
                "digest": attr.string(),
                "tag": attr.string(),
                "platforms": attr.string_list(),
            },
        ),
    },
)

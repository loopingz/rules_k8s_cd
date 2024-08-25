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
        _oci_push_info_sh = attr.label(allow_single_file = True, default = "//starlark:oci_push_info.sh.tpl"),
    ),
    toolchains = oci_push_lib.toolchains + ["@aspect_bazel_lib//lib:yq_toolchain_type"],
    executable = True,
    outputs = {
        "digest": "%{name}.digest",
    },
)

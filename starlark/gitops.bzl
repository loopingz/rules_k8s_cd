load("@rules_oci//oci/private:push.bzl", "oci_push_lib")
load("@com_adobe_rules_gitops//skylib:push.bzl", "K8sPushInfo")
load(
    "@com_adobe_rules_gitops//skylib/kustomize:kustomize.bzl",
    "kubectl",
    #"kustomize",
    kustomize_gitops = "gitops",
)
load("@com_adobe_rules_gitops//skylib:k8s.bzl", "show")
load("kustomize.bzl", "kustomize")

EnvironmentInfo = provider(fields = ["type"])

def _image_pushes(images, image_repository, name_suffix = ".push"):
    image_pushes = []

    def process_image(image_label, legacy_name = None):
        rule_name_parts = [image_repository, legacy_name]
        rule_name_parts = [p for p in rule_name_parts if p]
        rule_name = "_".join(rule_name_parts)
        rule_name = rule_name.replace("/", "_").replace(":", "_").replace("@", "_")
        if rule_name.startswith("_"):
            rule_name = rule_name[1:]
        rule_name = rule_name + name_suffix
        if not native.existing_rule(rule_name):
            oci_push_info(
                name = rule_name,
                image = image_label,  # buildifier: disable=uninitialized
                repository = image_repository + "/" + legacy_name,
            )
        return ":" + rule_name

    if type(images) == "dict":
        for image_name in images:
            image = images[image_name]
            push = process_image(image, image_name)
            image_pushes.append(push)
    else:
        for image in images:
            push = process_image(image)
            image_pushes.append(push)

    return image_pushes

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

    return [push_result, K8sPushInfo(
        image_label = "latest",
        registry = registry,
        repository = repository,
        digestfile = digestfile,
        legacy_image_name = ctx.attr.repository.split("/")[-1],
    )]

oci_push_info = rule(
    implementation = _oci_push_info,
    attrs = dict(
        oci_push_lib.attrs,
        _oci_push_info_sh = attr.label(allow_single_file = True, default = "//starlark:oci_push_info.sh.tpl"),
    ),
    toolchains = oci_push_lib.toolchains,
    executable = True,
    outputs = {
        "digest": "%{name}.digest",
    },
)

def _environment_impl(ctx):
    return EnvironmentInfo(type = ctx.label.name)

environment = rule(
    implementation = _environment_impl,
)

environments = [
   "dev",
   "beta",
   "preview",
   "user",
   "prod",
]

def k8s_deploy(
        name,  # name of the rule is important for gitops, since it will become a part of the target manifest file name in /cloud
        cluster = "dev",
        user = "{BUILD_USER}",
        namespace = None,
        configmaps_srcs = None,
        secrets_srcs = None,
        configmaps_renaming = None,  # configmaps renaming policy. Could be None or 'hash'.
        manifests = None,
        name_prefix = None,
        name_suffix = None,
        prefix_suffix_app_labels = False,  # apply kustomize configuration to modify "app" labels in Deployments when name prefix or suffix applied
        patches = None,
        image_name_patches = {},
        image_tag_patches = {},
        substitutions = {},  # dict of template parameter substitutions. CLUSTER and NAMESPACE parameters are added automatically.
        configurations = [],  # additional kustomize configuration files. rules_gitops provides
        common_labels = {},  # list of common labels to apply to all objects see commonLabels kustomize docs
        common_annotations = {},  # list of common annotations to apply to all objects see commonAnnotations kustomize docs
        deps = [],
        deps_aliases = {},
        images = [],
        image_digest_tag = False,
        image_registry = "docker.io",  # registry to push container to. jenkins will need an access configured for gitops to work. Ignored for mynamespace.
        image_repository = None,  # repository (registry path) to push container to. Generated from the image bazel path if empty.
        image_repository_prefix = None,  # Mutually exclusive with 'image_repository'. Add a prefix to the repository name generated from the image bazel path
        objects = [],
        gitops = True,  # make sure to use gitops = False to work with individual namespace. This option will be turned False if namespace is '{BUILD_USER}'
        gitops_path = "cloud",
        deployment_branch = None,
        release_branch_prefix = "main",
        start_tag = "{{",
        end_tag = "}}",
        visibility = None):
    """ k8s_deploy
    """

    if not manifests:
        manifests = native.glob(["*.yaml", "*.yaml.tpl"])
    if prefix_suffix_app_labels:
        configurations = configurations + [
            "@com_adobe_rules_gitops//skylib/kustomize:nameprefix_deployment_labels_config.yaml",
            "@com_adobe_rules_gitops//skylib/kustomize:namesuffix_deployment_labels_config.yaml",
        ]
    for reservedname in ["CLUSTER", "NAMESPACE"]:
        if substitutions.get(reservedname):
            fail("do not put %s in substitutions parameter of k8s_deploy. It will be added autimatically" % reservedname)
    substitutions = dict(substitutions)
    substitutions["CLUSTER"] = cluster
    substitutions["SOURCE"] = native.package_name() + ":" + name
    substitutions["ENVIRONMENT"] = name
    substitutions["GIT_COMMIT"] = "{STABLE_GIT_COMMIT}"
    #substitutions["BUILD_USER"] = "Test"

    annotations = dict(common_annotations)
    # annotations["com.loopingz.gitops.commit"] = "{STABLE_GIT_COMMIT}"
    # annotations["com.loopingz.gitops.target"] = native.package_name() + ":" + name

    #name_prefix = "{BUILD_USER}-"
    kustomize(
        name = name,
        namespace = namespace,
        configmaps_srcs = configmaps_srcs,
        secrets_srcs = secrets_srcs,
        # disable_name_suffix_hash is renamed to configmaps_renaming in recent Kustomize
        disable_name_suffix_hash = (configmaps_renaming != "hash"),
        images = images,
        manifests = manifests,
        substitutions = substitutions,
        deps = deps,
        deps_aliases = deps_aliases,
        start_tag = start_tag,
        end_tag = end_tag,
        name_prefix = name_prefix,
        name_suffix = name_suffix,
        configurations = configurations,
        common_labels = common_labels,
        common_annotations = annotations,
        patches = patches,
        objects = objects,
        image_name_patches = image_name_patches,
        image_tag_patches = image_tag_patches,
        visibility = visibility,
    )

    kubectl(
        name = name + ".apply",
        srcs = [name],
        cluster = cluster,
        user = "loopkube",
        namespace = namespace,
        visibility = visibility,
    )
    show(
        name = name + ".show",
        namespace = namespace,
        src = name,
        visibility = visibility,
    )
    if gitops:
        kustomize_gitops(
            name = name + ".gitops",
            srcs = [name],
            cluster = cluster,
            namespace = namespace,
            gitops_path = gitops_path,
            strip_prefixes = [
                namespace + "-",
                cluster + "-",
            ],
            deployment_branch = deployment_branch,
            release_branch_prefix = release_branch_prefix,
            visibility = ["//visibility:public"],
        )
    else:
        kubectl(
            name = name + ".delete",
            srcs = [name],
            command = "delete",
            cluster = cluster,
            push = False,
            user = user,
            namespace = namespace,
            visibility = visibility,
        )

def gitops(
        images,
        namespace = "bazel",
        image_registry = "docker.loopingz.com/bazel",
        manifests = [],
        **kwargs):
    image_pushes = _image_pushes(
        name_suffix = ".push",
        images = images,
        image_repository = image_registry,
    )

    for env in environments:
        if native.existing_rule(env):
            fail("Macro gitops should only be used once by BUILD file")
        k8s_deploy(
            name = env,
            images = image_pushes,
            cluster = "loopkube",
            namespace = "{BUILD_USER}" if env == "user" else namespace + "-" + env,
            gitops = env in ["prod", "beta", "preview"],
            image_registry = image_registry,
            manifests = native.glob([
                "manifests/*",
                "manifests/%s/**/*" % env,
            ]),
            patches = native.glob([
                "overlays/%s/**/*.yaml" % env,
            ]),
            **kwargs
        )

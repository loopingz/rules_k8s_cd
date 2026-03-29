# Rule that validates Kubernetes manifests against Kyverno policies.
def _kyverno_impl(ctx):
    kyverno_bin = ctx.toolchains["@rules_k8s_cd//lib:kyverno_toolchain_type"].kyvernoinfo.bin
    command = [kyverno_bin.short_path, "apply"]

    for f in ctx.files.policies:
        command.append(f.short_path)

    for f in ctx.files.manifests:
        command.append("-r")
        command.append(f.short_path)

    for f in ctx.files.exceptions:
        command.append("-e")
        command.append(f.short_path)

    command.append("--detailed-results")

    cmd = " ".join(command)

    ctx.actions.write(
        output = ctx.outputs.test,
        content = cmd,
    )

    return [DefaultInfo(
        executable = ctx.outputs.test,
        runfiles = ctx.runfiles(files = [
            kyverno_bin,
        ] + ctx.files.policies + ctx.files.manifests + ctx.files.exceptions),
    )]

kyverno_test = rule(
    implementation = _kyverno_impl,
    doc = "Validates Kubernetes manifests against Kyverno policies.",
    attrs = {
        "policies": attr.label_list(
            mandatory = True,
            allow_files = [".yaml"],
            doc = "Kyverno policy YAML files to validate against.",
        ),
        "manifests": attr.label_list(
            mandatory = True,
            allow_files = [".yaml"],
            doc = "Kubernetes manifest files to validate.",
        ),
        "exceptions": attr.label_list(
            mandatory = False,
            allow_files = [".yaml"],
            doc = "Kyverno policy exception YAML files.",
        ),
    },
    toolchains = ["@rules_k8s_cd//lib:kyverno_toolchain_type"],
    outputs = {"test": "%{name}.sh"},
    test = True,
    executable = False,
)

def kyverno_baseline(name, manifests, exceptions = [], **kwargs):
    """Validates manifests against Kubernetes Pod Security Standards - Baseline profile.

    Args:
        name: Target name.
        manifests: List of Kubernetes manifest files to validate.
        exceptions: Optional list of Kyverno policy exception files.
        **kwargs: Additional arguments passed to kyverno_test.
    """
    kyverno_test(
        name = name,
        policies = ["@rules_k8s_cd//lib/kyverno/policies:baseline"],
        manifests = manifests,
        exceptions = exceptions,
        **kwargs
    )

def kyverno_restricted(name, manifests, exceptions = [], **kwargs):
    """Validates manifests against Kubernetes Pod Security Standards - Restricted profile.

    Restricted is a superset of Baseline, so all baseline policies are included.

    Args:
        name: Target name.
        manifests: List of Kubernetes manifest files to validate.
        exceptions: Optional list of Kyverno policy exception files.
        **kwargs: Additional arguments passed to kyverno_test.
    """
    kyverno_test(
        name = name,
        policies = ["@rules_k8s_cd//lib/kyverno/policies:restricted"],
        manifests = manifests,
        exceptions = exceptions,
        **kwargs
    )

def kyverno_best_practices(name, manifests, exceptions = [], **kwargs):
    """Validates manifests against operational best practices.

    Checks for required labels, resource limits/requests, and disallows latest tag.

    Args:
        name: Target name.
        manifests: List of Kubernetes manifest files to validate.
        exceptions: Optional list of Kyverno policy exception files.
        **kwargs: Additional arguments passed to kyverno_test.
    """
    kyverno_test(
        name = name,
        policies = ["@rules_k8s_cd//lib/kyverno/policies:best-practices"],
        manifests = manifests,
        exceptions = exceptions,
        **kwargs
    )

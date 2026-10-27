load("@rules_shell//shell:sh_test.bzl", "sh_test")
load("//lib:kyverno.bzl", "kyverno_baseline", "kyverno_test")

# The pinned CLI evaluates both policy APIs: CEL ValidatingPolicy (autogen to the Deployment) ...
kyverno_test(
    name = "validating_policy_test",
    manifests = ["compliant.yaml"],
    policies = ["validating-policy.yaml"],
)

# ... and the legacy ClusterPolicy bundles shipped in lib/kyverno/policies
kyverno_baseline(
    name = "baseline_test",
    manifests = ["compliant.yaml"],
)

# A violation fails the test: run the generated test script on a violating manifest and expect a failure
kyverno_test(
    name = "violating",
    manifests = ["violating.yaml"],
    policies = ["validating-policy.yaml"],
    tags = ["manual"],
)

sh_test(
    name = "violation_fails_test",
    srcs = ["expect_failure.sh"],
    args = ["$(rootpath :violating)"],
    data = [":violating"],
)

# rules_k8s_cd — Bazel rules for Kubernetes continuous delivery

Bazel rules for building, deploying, scanning, and managing Kubernetes resources using kubectl, kustomize, and OCI images.

## Rules

`kubectl`: Run kubectl commands via `bazel run`

`kubectl_export`: Capture kubectl output to a file at build time

`grype_scan`: Scan OCI images for vulnerabilities using Grype

`trivy_scan`: Scan OCI images for vulnerabilities using Trivy

`trivy_sbom`: Generate an SBOM for an OCI image using Trivy

`kyverno_test`: Validate Kubernetes manifests against Kyverno policies

### Kyverno Policy Presets

Three convenience macros validate manifests against curated policy sets:

`kyverno_baseline`: Validates against [Kubernetes Pod Security Standards - Baseline](https://kubernetes.io/docs/concepts/security/pod-security-standards/#baseline) (disallow privileged containers, host namespaces, host ports)

`kyverno_restricted`: Validates against [Kubernetes Pod Security Standards - Restricted](https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted) (all baseline + disallow privilege escalation, require run as non-root)

`kyverno_best_practices`: Validates against operational best practices (require labels, resource limits, disallow latest tag)

Usage:

```python
load("@rules_k8s_cd//lib:kyverno.bzl", "kyverno_baseline", "kyverno_best_practices", "kyverno_restricted")

kyverno_baseline(
    name = "security_baseline",
    manifests = glob(["*.yaml"]),
)

kyverno_restricted(
    name = "security_restricted",
    manifests = glob(["*.yaml"]),
)

kyverno_best_practices(
    name = "best_practices",
    manifests = glob(["*.yaml"]),
)
```

### Importing Kyverno Policies

Import policies from the [official Kyverno policy library](https://kyverno.io/policies/):

```bash
bazel run @rules_k8s_cd//go/kyverno_import -- best-practices/require-labels
bazel run @rules_k8s_cd//go/kyverno_import -- pod-security/restricted/disallow-privilege-escalation
```

Configure the output directory in `MODULE.bazel`:

```python
bazel_lib_toolchains.kyverno(
    policies_dir = "infra/policies",  # default: "kyverno/policies"
)
```

`dive`: Analyze OCI image layers using Dive

## Macros

`kustomize`: Build a kustomize overlay and export the rendered YAML

`kustomize_show`: Display the rendered kustomize output via `bazel run`

`kustomize_apply`: Apply a kustomize overlay to a cluster via `bazel run`

`kustomize_gitops`: Generate kustomize output for gitops workflows (writes rendered YAML back to the repo)

`kustomization_injector`: Rule that injects images and patches into a kustomization.yaml

## Utils

Rules:

- `run_all`: Run all executable dependencies before running the command itself
- `show`: Display files of another target
- `write_source_files`: Write multiple generated files back to the repository
- `write_source_file`: Write one generated file back to the repository
- `tar_filter`: Filter entries from a tar archive

Repository rules:

- `download_binary`: Repository rule to download a binary from a url

## Usage

```bash
# To see all the targets
bazel query //deployments/...
```

Result

```bash
//deployments/website:beta
//deployments/website:beta.apply
//deployments/website:beta.gitops
//deployments/website:beta.show
//deployments/website:dev
//deployments/website:dev.apply
//deployments/website:dev.delete
//deployments/website:dev.show
//deployments/website:docker.loopingz.com_bazel_website.push
//deployments/website:preview
//deployments/website:preview.apply
//deployments/website:preview.gitops
//deployments/website:preview.show
//deployments/website:prod
//deployments/website:prod.apply
//deployments/website:prod.gitops
//deployments/website:prod.show
//deployments/website:user
//deployments/website:user.apply
//deployments/website:user.delete
//deployments/website:user.show
```

When you deploy the main resources have labels:

Like this service:

```
apiVersion: v1
kind: Service
metadata:
  annotations:
    gitops.loopingz.com/target: deployments/website:preview
  labels:
    app: website
    gitops.loopingz.com/commit: 09f7b66
    gitops.loopingz.com/environment: preview
```

It is then easy to clean-up with a command like this:

```
kubectl -n bazel-preview delete all -l 'gitops.loopingz.com/commit,gitops.loopingz.com/commit!=09f7b66,gitops.loopingz.com/environment=preview'
```

## Use the oci extension

The oci_plus extension is a wrapper around oci_pull that automatically creates augmented repositories for container images. It makes the enhanced repo the primary entrypoint (@<name>), while keeping the raw oci_pull repo available as @<name>_original.

What it does

For each image you declare:
	- Runs oci_pull under the hood as <name>_original.
	- Creates a companion repo <name> that:
	- Aliases the base image.
	- Exposes an oci_load target for loading into Docker/CRI.
	- Adds vulnerability scan rules (grype_scan, trivy_scan) and an SBOM generator (trivy_sbom), and dive utility (dive)


```MODULE.bazel
bazel_dep(name = "rules_oci", version = ">=2.2.0")
bazel_dep(name = "rules_k8s_cd", version = "...")  # where your scan rules live

oci_plus = use_extension("//tools/oci_ext:extension.bzl", "oci_plus")

oci_plus.pull(
    name = "renovate",
    image = "ghcr.io/renovatebot/renovate",
    digest = "sha256:...",   # or tag = "x.y.z"
    platforms = ["linux/amd64", "linux/arm64/v8"],
)

# Import both repos so they're visible to your build
use_repo(oci_plus, "renovate")
```

Targets available

In the enhanced repo (@renovate):
 - @renovate//:base -> alias to @renovate_original//:renovate
 - @renovate//:dive -> explore the image
 - @renovate//:load -> oci_load target (loads into Docker/CRI)
 - @renovate//:grype -> run Grype scan
 - @renovate//:trivy -> run Trivy scan
 - @renovate//:sbom -> generate a Trivy SBOM


## How to use grype

You can use the grype scanner from your bazel installation directly with:

```
bazel run @grype_bin//:grype -- --help
```

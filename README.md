# rules_k8s_cd bazel rules for kubernetes development

This repository includes a Bazel empty directory for monorepo with gitops.

## Rules

`kubectl`: Allow you to launch kubectl commands with bazel

`kubectl_export`: Allow you to export kubectl command to a file with bazel, the kubeconfig is currently empty so this won't allow you to connect to a cluster [GitHub Issue](https://github.com/loopingz/rules_k8s_cd/issues/1)

`dive`: Dive into your image

`grype_scan`: Scan your image with nice scanner

## Macros

`kustomize`: Launch kubectl with kustomize

`gitops`: Deploy your application with gitops (kustomize, helm, kubectl, terraform, ...)
This macro is more an example to fork.

## Utils

Rules:

- `run_all`: Run all executable dependencies before running the command itself
- `show`: Display files of another target
- `write_source_files`: Allow you to write some generated files back to the repository
- `write_source_file`: Allow you to write some one file back to the repository

Repository rules:

- `download_toolchain_binary`: Repository rule to download a binary from a url

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
	•	Runs oci_pull under the hood as <name>_original.
	•	Creates a companion repo <name> that:
	•	Aliases the base image.
	•	Exposes an oci_load target for loading into Docker/CRI.
	•	Adds vulnerability scan rules (grype_scan, trivy_scan) and an SBOM generator (trivy_sbom), and dive utility (dive)


```MODULE.bazel
bazel_dep(name = "rules_oci", version = ">=2.2.0")
bazel_dep(name = "rules_k8s_cd", version = "…")  # where your scan rules live

oci_plus = use_extension("//tools/oci_ext:extension.bzl", "oci_plus")

oci_plus.pull(
    name = "renovate",
    image = "ghcr.io/renovatebot/renovate",
    digest = "sha256:…",   # or tag = "x.y.z"
    platforms = ["linux/amd64", "linux/arm64/v8"],
)

# Import both repos so they’re visible to your build
use_repo(oci_plus, "renovate")
```

Targets available

In the enhanced repo (@renovate):
 - @renovate//:base → alias to @renovate_original//:renovate
 - @renovate//:dive → explore the image
 - @renovate//:load → oci_load target (loads into Docker/CRI)
 - @renovate//:grype → run Grype scan
 - @renovate//:trivy → run Trivy scan
 - @renovate//:sbom → generate a Trivy SBOM


## How to use grype

You can use the grype scanner from your bazel installation directly with:

```
bazel run @grype_bin//:grype -- --help
```

# rules_k8s_cd

Kubernets Continuous Deployment rules for Bazel

This repository contains rules for building and deploying Kubernetes applications with Bazel.
It also contains some rules for IaC.

## Rules

`kubectl`: Allow you to launch kubectl commands with bazel
`kubectl_export`: Allow you to export kubectl command to a file with bazel
`helm`: Helm export
`grype`: Scan your image with nice scanner
`terraform`: Deploy your terraform with bazel

## Macros

`kustomize`: Launch kubectl with kustomize
`gitops`: Deploy your application with gitops (kustomize, helm, kubectl, terraform, ...)

## Utils

Rules:

- `run_all`: Run all executable dependencies before running the command itself
- `show`: Display files of another target
- `write_source_files`: Allow you to write some generated files back to the repository

Repository rules:

 - `download_binary`: Repository rule to download a binary from a url

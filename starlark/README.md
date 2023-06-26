# rules_k8s_cd

Kubernetes Continuous Deployment rules for Bazel

This repository contains rules for building and deploying Kubernetes applications with Bazel.
It also contains some rules for IaC.

## Rules

`kubectl`: Allow you to launch kubectl commands with bazel

`kubectl_export`: Allow you to export kubectl command to a file with bazel, the kubeconfig is currently empty so this won't allow you to connect to a cluster [GitHub Issue](https://github.com/loopingz/rules_k8s_cd/issues/1)

`helm`: Helm export - TO BE IMPLEMENTED

`grype_scan`: Scan your image with nice scanner

`terraform`: Deploy your terraform with bazel

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

 - `download_binary`: Repository rule to download a binary from a url

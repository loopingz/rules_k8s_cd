# rules_k8s_cd

Kubernets Continuous Deployment rules for Bazel

This repository contains rules for building and deploying Kubernetes applications with Bazel.
It also contains soem rules for IaC.

## Rules

`kubectl`: Allow you to launch kubectl commands with bazel
`helm`: Helm export
`grype`: Scan your image with nice scanner
`gitops`: Deploy your application with gitops (kustomize, helm, kubectl, terraform, ...)
`terraform`: Deploy your terraform with bazel

## Utils

Rules:

- `run_all`: Run all executable dependencies before running the command itself
- `show`: Display files of another target

Repository rules:

 - `download_binary`: Repository rule to download a binary from a url

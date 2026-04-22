# Helm rules

## Overview

The Helm rules let you pull charts from classic HTTP Helm repositories and render them as Bazel build outputs. `helm_pull` is a module extension that downloads and pins a chart tarball at analysis time, exposing it as a filegroup label. `helm_template` is a rule (backed by a Helm 3 toolchain) that runs `helm template`, then post-processes the output: splitting it into individual resource documents, dropping unwanted resources via a declarative filter, and applying strategic-merge and/or JSON6902 patches before writing a single consolidated YAML file.

---

## `helm_pull` module extension

Declare charts in `MODULE.bazel`:

```python
helm_pull = use_extension("@rules_k8s_cd//lib:helm.bzl", "helm_pull")

helm_pull.chart(
    name       = "ingress_nginx",
    repo       = "https://kubernetes.github.io/ingress-nginx",
    chart      = "ingress-nginx",
    version    = "4.10.0",
    sha256     = "abc123...",   # omit on first run to discover the value
)

use_repo(helm_pull, "ingress_nginx")
```

### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Repository name used to reference the chart (e.g. `@ingress_nginx`). |
| `repo` | string | yes | Base URL of the Helm repository. |
| `chart` | string | yes | Chart name as published in the repository index. |
| `version` | string | yes | Exact chart version to download. |
| `sha256` | string | no | Expected SHA-256 of the downloaded tarball. Strongly recommended for reproducible builds. |
| `url` | string | no | Explicit tarball URL. Overrides the default `<repo>/<chart>-<version>.tgz` construction. |

### Targets exposed by the generated repository

| Target | Description |
|--------|-------------|
| `@<name>//:chart` | `filegroup` containing all files in the chart directory — pass this to `helm_template`'s `chart` attribute. |
| `@<name>//:sha256` | Runnable `sh_binary` that prints `sha256 = "<value>"` — useful for pinning a new chart. |
| `@<name>//:sha256.txt` | Plain text file containing the computed SHA-256. |

### SHA-256 pinning workflow

On the first pull, omit `sha256`. Bazel will print a warning:

```
WARNING: helm_pull.chart(name = "ingress_nginx") has no sha256 pinned.
         Computed: sha256 = "abc123..."
         Add this to MODULE.bazel for reproducibility.
```

Copy the value into `MODULE.bazel` and run the build again. Alternatively:

```bash
bazel run @ingress_nginx//:sha256
```

---

## `helm_template` rule

```python
load("@rules_k8s_cd//lib:helm.bzl", "helm_template")
```

Renders a Helm chart, applies post-processing, and writes a single YAML file.

### Attributes

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `name` | Name | yes | — | Target name. |
| `chart` | label | yes | — | Chart directory label. Typically `@<name>//:chart` from `helm_pull`, or a local `filegroup` containing `Chart.yaml` and all chart files. |
| `values` | list of labels | no | `[]` | Values YAML files merged in order (later files take precedence, equivalent to `-f` flags). |
| `release_name` | string | no | target name | Helm release name passed to `helm template`. Defaults to the Bazel target name. |
| `namespace` | string | no | `"default"` | Kubernetes namespace for rendering (`--namespace`). |
| `exclude` | list of dicts | no | `[]` | Resource selectors; any resource matching a selector is removed from the output. See [Filter syntax](#filter-syntax). |
| `patchesStrategicMerge` | list of labels | no | `[]` | Strategic-merge patch YAML files. See [Patch file formats](#patch-file-formats). |
| `patchesJson6902` | list of labels | no | `[]` | JSON6902 patch YAML files. See [Patch file formats](#patch-file-formats). |
| `out` | output | no | `<name>.yaml` | Output YAML file. Defaults to `<name>.yaml`. |

The rule requires the `@rules_k8s_cd//lib:helm_toolchain_type` toolchain (Helm 3.16.4 by default).

---

## Filter syntax

```python
exclude = [
    {"kind": "NetworkPolicy"},
    {"kind": "PodDisruptionBudget", "namespace": "kube-system"},
]
```

Each dict in the `exclude` list is a **selector**. The keys available are:

| Key | Description |
|-----|-------------|
| `apiVersion` | Match `apiVersion` field of the resource. |
| `kind` | Match `kind` field of the resource. |
| `name` | Match `metadata.name` of the resource. |
| `namespace` | Match `metadata.namespace` of the resource. |

**AND within a selector, OR across selectors.** A resource is dropped when it matches *all* keys of *any* selector dict.

Examples:

```python
# Drop all NetworkPolicy resources (any namespace)
exclude = [{"kind": "NetworkPolicy"}]

# Drop a specific resource by kind and name
exclude = [{"kind": "ServiceAccount", "name": "ingress-nginx-admission"}]

# Drop either of two resource types
exclude = [
    {"kind": "NetworkPolicy"},
    {"kind": "PodDisruptionBudget"},
]
```

---

## Patch file formats

### Strategic-merge patch

A strategic-merge patch is a partial resource document. The `apiVersion`, `kind`, and `metadata.name` fields (plus optionally `metadata.namespace`) identify the target resource; all other fields are merged in.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
        - name: my-app
          resources:
            limits:
              cpu: "500m"
              memory: "256Mi"
```

Well-known core Kubernetes kinds (Pod, Service, ConfigMap, Secret, PersistentVolumeClaim) use proper strategic-merge semantics (lists are merged by strategic key). All other kinds fall back to RFC 7396 JSON merge patch, where list fields are replaced wholesale.

### JSON6902 patch

A JSON6902 patch file contains two top-level keys: `target` (identifying the resource) and `patch` (a list of [RFC 6902](https://datatracker.ietf.org/doc/html/rfc6902) operations).

```yaml
target:
  apiVersion: apps/v1
  kind: Deployment
  name: my-app
  # namespace: optional — omit for cluster-scoped resources or namespace-agnostic matching
patch:
  - op: replace
    path: /spec/replicas
    value: 3
  - op: add
    path: /metadata/labels/team
    value: platform
  - op: remove
    path: /spec/template/spec/automountServiceAccountToken
```

Supported operations: `add`, `remove`, `replace`, `move`, `copy`, `test`.

---

## End-to-end example

This mirrors the fixture in `example/deployments/helm_demo/`.

### `MODULE.bazel`

```python
bazel_dep(name = "rules_k8s_cd", version = "...")

helm_pull = use_extension("@rules_k8s_cd//lib:helm.bzl", "helm_pull")

helm_pull.chart(
    name    = "ingress_nginx",
    repo    = "https://kubernetes.github.io/ingress-nginx",
    chart   = "ingress-nginx",
    version = "4.10.0",
    sha256  = "abc123...",
)

use_repo(helm_pull, "ingress_nginx")
```

### `deployments/ingress_nginx/BUILD`

```python
load("@rules_k8s_cd//lib:helm.bzl", "helm_template")

helm_template(
    name         = "ingress_nginx_prod",
    # Chart files fetched and pinned by helm_pull
    chart        = "@ingress_nginx//:chart",
    # Values files merged in order; later files win
    values       = [
        "values.yaml",
        "values.prod.yaml",
    ],
    release_name = "ingress-nginx",
    namespace    = "ingress-system",
    # Remove resources that conflict with cluster-level policies
    exclude = [
        {"kind": "NetworkPolicy"},
        {"kind": "PodDisruptionBudget"},
    ],
    # Increase resource limits for production
    patchesStrategicMerge = ["patches/resources.yaml"],
    # Add a toleration to the controller Deployment
    patchesJson6902       = ["patches/tolerations.yaml"],
    # Output written to ingress_nginx_prod.yaml
)
```

**What each piece does:**

- `chart = "@ingress_nginx//:chart"` — refers to the pinned tarball from `helm_pull`; Bazel will re-download only when `version` or `sha256` changes.
- `values` — same as passing multiple `-f` flags to `helm template`; the second file overrides keys in the first.
- `exclude` — post-render filter; Helm still renders all resources, but the filtered kinds are removed before the output is written.
- `patchesStrategicMerge` / `patchesJson6902` — applied after filtering; use strategic-merge for additive container-spec changes and JSON6902 for precise field operations (replace a value, add a label, remove a field).

### Toolchain setup

```python
# MODULE.bazel
bazel_lib_toolchains = use_extension("@rules_k8s_cd//lib:extensions.bzl", "toolchains")
bazel_lib_toolchains.helm()
use_repo(bazel_lib_toolchains, "helm", "helm_toolchains")

register_toolchains("@helm_toolchains//:all")
```

---

## Known limitations

The `helm_template` rule depends on `//go/helm_postrender`, which currently requires additional Bazel-Go vendor BUILD file setup for `k8s.io/*` dependencies that has not yet landed. Track the follow-up in the project issue tracker (placeholder link). The Go code itself is complete and passes `go test`; only the Bazel vendoring wiring remains.

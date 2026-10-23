<!-- Generated with Stardoc: http://skydoc.bazel.build -->

Render local Helm charts as Kubernetes YAML, optionally excluding resources.

<a id="helm_template"></a>

## helm_template

<pre>
load("@rules_k8s_cd//lib:helm.bzl", "helm_template")

helm_template(<a href="#helm_template-name">name</a>, <a href="#helm_template-data">data</a>, <a href="#helm_template-out">out</a>, <a href="#helm_template-api_versions">api_versions</a>, <a href="#helm_template-chart">chart</a>, <a href="#helm_template-exclude_kinds">exclude_kinds</a>, <a href="#helm_template-exclude_resources">exclude_resources</a>, <a href="#helm_template-include_crds">include_crds</a>,
              <a href="#helm_template-kube_version">kube_version</a>, <a href="#helm_template-namespace">namespace</a>, <a href="#helm_template-release_name">release_name</a>, <a href="#helm_template-set">set</a>, <a href="#helm_template-set_string">set_string</a>, <a href="#helm_template-skip_tests">skip_tests</a>, <a href="#helm_template-values">values</a>)
</pre>

Render a local Helm chart at build time and remove selected resources. Dependencies must be vendored; rendering does not fetch charts or use local Helm configuration.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="helm_template-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="helm_template-data"></a>data |  All chart files, including templates, defaults, and vendored dependencies. Use glob(["chart/**"]).   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="helm_template-out"></a>out |  Output YAML; defaults to <name>.yaml.   | <a href="https://bazel.build/concepts/labels">Label</a>; <a href="https://bazel.build/reference/be/common-definitions#configurable-attributes">nonconfigurable</a> | optional |  `None`  |
| <a id="helm_template-api_versions"></a>api_versions |  Additional API versions available in Helm capabilities.   | List of strings | optional |  `[]`  |
| <a id="helm_template-chart"></a>chart |  Chart.yaml, a local chart archive, or a generated directory containing the chart.   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="helm_template-exclude_kinds"></a>exclude_kinds |  Exact, case sensitive kinds to remove, e.g. ["Secret", "Job"].   | List of strings | optional |  `[]`  |
| <a id="helm_template-exclude_resources"></a>exclude_resources |  Case sensitive glob patterns Kind/name (all namespaces) or Kind/namespace/name. Each component supports *, ?, and character classes; an empty namespace matches omitted metadata.namespace.   | List of strings | optional |  `[]`  |
| <a id="helm_template-include_crds"></a>include_crds |  Include files from the chart's crds directory.   | Boolean | optional |  `False`  |
| <a id="helm_template-kube_version"></a>kube_version |  Kubernetes version used for Helm capabilities.   | String | optional |  `""`  |
| <a id="helm_template-namespace"></a>namespace |  Release namespace. Filters match metadata.namespace as rendered, including when it is omitted.   | String | optional |  `""`  |
| <a id="helm_template-release_name"></a>release_name |  Release name; defaults to the target name.   | String | optional |  `""`  |
| <a id="helm_template-set"></a>set |  Helm --set overrides; Helm interprets value types and escaping.   | <a href="https://bazel.build/rules/lib/core/dict">Dictionary: String -> String</a> | optional |  `{}`  |
| <a id="helm_template-set_string"></a>set_string |  Helm --set-string overrides, preserving string values.   | <a href="https://bazel.build/rules/lib/core/dict">Dictionary: String -> String</a> | optional |  `{}`  |
| <a id="helm_template-skip_tests"></a>skip_tests |  Omit Helm test hooks.   | Boolean | optional |  `False`  |
| <a id="helm_template-values"></a>values |  Values files, applied in order; later files take precedence.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |



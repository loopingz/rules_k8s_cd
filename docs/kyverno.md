<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="kyverno_test"></a>

## kyverno_test

<pre>
load("@rules_k8s_cd//lib:kyverno.bzl", "kyverno_test")

kyverno_test(<a href="#kyverno_test-name">name</a>, <a href="#kyverno_test-exceptions">exceptions</a>, <a href="#kyverno_test-manifests">manifests</a>, <a href="#kyverno_test-policies">policies</a>)
</pre>

Validates Kubernetes manifests against Kyverno policies.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="kyverno_test-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="kyverno_test-exceptions"></a>exceptions |  Kyverno policy exception YAML files.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="kyverno_test-manifests"></a>manifests |  Kubernetes manifest files to validate.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | required |  |
| <a id="kyverno_test-policies"></a>policies |  Kyverno policy YAML files to validate against.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | required |  |


<a id="kyverno_baseline"></a>

## kyverno_baseline

<pre>
load("@rules_k8s_cd//lib:kyverno.bzl", "kyverno_baseline")

kyverno_baseline(<a href="#kyverno_baseline-name">name</a>, <a href="#kyverno_baseline-manifests">manifests</a>, <a href="#kyverno_baseline-exceptions">exceptions</a>, <a href="#kyverno_baseline-kwargs">**kwargs</a>)
</pre>

Validates manifests against Kubernetes Pod Security Standards - Baseline profile.

**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="kyverno_baseline-name"></a>name |  Target name.   |  none |
| <a id="kyverno_baseline-manifests"></a>manifests |  List of Kubernetes manifest files to validate.   |  none |
| <a id="kyverno_baseline-exceptions"></a>exceptions |  Optional list of Kyverno policy exception files.   |  `[]` |
| <a id="kyverno_baseline-kwargs"></a>kwargs |  Additional arguments passed to kyverno_test.   |  none |


<a id="kyverno_best_practices"></a>

## kyverno_best_practices

<pre>
load("@rules_k8s_cd//lib:kyverno.bzl", "kyverno_best_practices")

kyverno_best_practices(<a href="#kyverno_best_practices-name">name</a>, <a href="#kyverno_best_practices-manifests">manifests</a>, <a href="#kyverno_best_practices-exceptions">exceptions</a>, <a href="#kyverno_best_practices-kwargs">**kwargs</a>)
</pre>

Validates manifests against operational best practices.

Checks for required labels, resource limits/requests, and disallows latest tag.


**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="kyverno_best_practices-name"></a>name |  Target name.   |  none |
| <a id="kyverno_best_practices-manifests"></a>manifests |  List of Kubernetes manifest files to validate.   |  none |
| <a id="kyverno_best_practices-exceptions"></a>exceptions |  Optional list of Kyverno policy exception files.   |  `[]` |
| <a id="kyverno_best_practices-kwargs"></a>kwargs |  Additional arguments passed to kyverno_test.   |  none |


<a id="kyverno_restricted"></a>

## kyverno_restricted

<pre>
load("@rules_k8s_cd//lib:kyverno.bzl", "kyverno_restricted")

kyverno_restricted(<a href="#kyverno_restricted-name">name</a>, <a href="#kyverno_restricted-manifests">manifests</a>, <a href="#kyverno_restricted-exceptions">exceptions</a>, <a href="#kyverno_restricted-kwargs">**kwargs</a>)
</pre>

Validates manifests against Kubernetes Pod Security Standards - Restricted profile.

Restricted is a superset of Baseline, so all baseline policies are included.


**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="kyverno_restricted-name"></a>name |  Target name.   |  none |
| <a id="kyverno_restricted-manifests"></a>manifests |  List of Kubernetes manifest files to validate.   |  none |
| <a id="kyverno_restricted-exceptions"></a>exceptions |  Optional list of Kyverno policy exception files.   |  `[]` |
| <a id="kyverno_restricted-kwargs"></a>kwargs |  Additional arguments passed to kyverno_test.   |  none |



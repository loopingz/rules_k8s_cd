<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="trivy_sbom"></a>

## trivy_sbom

<pre>
load("@rules_k8s_cd//lib:trivy.bzl", "trivy_sbom")

trivy_sbom(<a href="#trivy_sbom-name">name</a>, <a href="#trivy_sbom-srcs">srcs</a>, <a href="#trivy_sbom-images">images</a>, <a href="#trivy_sbom-manifests">manifests</a>)
</pre>

Rule that generates an SBOM (Software Bill of Materials) using Trivy.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="trivy_sbom-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="trivy_sbom-srcs"></a>srcs |  List of inputs. The test will scan all images passed as srcs.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="trivy_sbom-images"></a>images |  List of images. The test will scan all images passed as srcs.   | List of strings | optional |  `[]`  |
| <a id="trivy_sbom-manifests"></a>manifests |  List of manifests. The test will scan all images defined inside manifests.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |


<a id="trivy_scan"></a>

## trivy_scan

<pre>
load("@rules_k8s_cd//lib:trivy.bzl", "trivy_scan")

trivy_scan(<a href="#trivy_scan-name">name</a>, <a href="#trivy_scan-srcs">srcs</a>, <a href="#trivy_scan-images">images</a>, <a href="#trivy_scan-manifests">manifests</a>)
</pre>

Rule that scans container images for vulnerabilities using Trivy.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="trivy_scan-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="trivy_scan-srcs"></a>srcs |  List of inputs. The test will scan all images passed as srcs.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="trivy_scan-images"></a>images |  List of images. The test will scan all images passed as srcs.   | List of strings | optional |  `[]`  |
| <a id="trivy_scan-manifests"></a>manifests |  List of manifests. The test will scan all images defined inside manifests.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |



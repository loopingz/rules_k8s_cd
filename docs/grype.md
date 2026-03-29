<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="grype_scan"></a>

## grype_scan

<pre>
load("@rules_k8s_cd//lib:grype.bzl", "grype_scan")

grype_scan(<a href="#grype_scan-name">name</a>, <a href="#grype_scan-srcs">srcs</a>, <a href="#grype_scan-images">images</a>, <a href="#grype_scan-manifests">manifests</a>)
</pre>

Rule that scans container images for vulnerabilities using Grype.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="grype_scan-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="grype_scan-srcs"></a>srcs |  List of inputs. The test will scan all images passed as srcs.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="grype_scan-images"></a>images |  List of images. The test will scan all images passed as srcs.   | List of strings | optional |  `[]`  |
| <a id="grype_scan-manifests"></a>manifests |  List of manifests. The test will scan all images defined inside manifests.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |



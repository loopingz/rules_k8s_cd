<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="dive"></a>

## dive

<pre>
load("@rules_k8s_cd//lib:dive.bzl", "dive")

dive(<a href="#dive-name">name</a>, <a href="#dive-srcs">srcs</a>, <a href="#dive-images">images</a>, <a href="#dive-manifests">manifests</a>)
</pre>



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="dive-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="dive-srcs"></a>srcs |  List of inputs. The test will scan all images passed as srcs.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="dive-images"></a>images |  List of images. The test will scan all images passed as srcs.   | List of strings | optional |  `[]`  |
| <a id="dive-manifests"></a>manifests |  List of manifests. The test will scan all images defined inside manifests.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |



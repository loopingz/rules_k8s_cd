<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="run_all"></a>

## run_all

<pre>
load("@rules_k8s_cd//lib:utils.bzl", "run_all")

run_all(<a href="#run_all-name">name</a>, <a href="#run_all-delimiter">delimiter</a>, <a href="#run_all-parallel">parallel</a>, <a href="#run_all-targets">targets</a>, <a href="#run_all-wrap_exits">wrap_exits</a>)
</pre>



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="run_all-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="run_all-delimiter"></a>delimiter |  -   | String | optional |  `""`  |
| <a id="run_all-parallel"></a>parallel |  -   | Boolean | optional |  `True`  |
| <a id="run_all-targets"></a>targets |  -   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="run_all-wrap_exits"></a>wrap_exits |  -   | Boolean | optional |  `False`  |


<a id="show"></a>

## show

<pre>
load("@rules_k8s_cd//lib:utils.bzl", "show")

show(<a href="#show-name">name</a>, <a href="#show-src">src</a>, <a href="#show-content">content</a>)
</pre>



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="show-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="show-src"></a>src |  Input file(s).   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="show-content"></a>content |  -   | Boolean | optional |  `False`  |


<a id="tar_filter"></a>

## tar_filter

<pre>
load("@rules_k8s_cd//lib:utils.bzl", "tar_filter")

tar_filter(<a href="#tar_filter-name">name</a>, <a href="#tar_filter-srcs">srcs</a>, <a href="#tar_filter-filters">filters</a>, <a href="#tar_filter-substitutions">substitutions</a>, <a href="#tar_filter-verbose">verbose</a>)
</pre>

tar_filter creates a new tar.gz file from the input tar.gz file(s) with the specified filters and substitutions.

You can use it to remove files from a tar.gz file, or to modify the paths of files in a tar.gz file.

Example:

        tar_filter(
            name = "filtered",
            srcs = [":original"],
            filters = [
                '^(\./)?usr',  # Remove all files in the usr directory
            ],
            substitutions = [
                '^\./:',  # Remove leading ./ from paths
                '^bin/:usr/bin/',  # Move all files in bin to usr/bin
            ],
        )

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="tar_filter-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="tar_filter-srcs"></a>srcs |  Input file(s).   | <a href="https://bazel.build/concepts/labels">List of labels</a> | required |  |
| <a id="tar_filter-filters"></a>filters |  -   | List of strings | optional |  `["^(\\./)?usr/share/man/", "^(\\./)?usr/share/doc/"]`  |
| <a id="tar_filter-substitutions"></a>substitutions |  -   | List of strings | optional |  `["^\\./:"]`  |
| <a id="tar_filter-verbose"></a>verbose |  -   | Boolean | optional |  `False`  |


<a id="write_source_file"></a>

## write_source_file

<pre>
load("@rules_k8s_cd//lib:utils.bzl", "write_source_file")

write_source_file(<a href="#write_source_file-name">name</a>, <a href="#write_source_file-src">src</a>, <a href="#write_source_file-target">target</a>)
</pre>



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="write_source_file-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="write_source_file-src"></a>src |  Input file(s).   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="write_source_file-target"></a>target |  -   | String | required |  |


<a id="write_source_files"></a>

## write_source_files

<pre>
load("@rules_k8s_cd//lib:utils.bzl", "write_source_files")

write_source_files(<a href="#write_source_files-name">name</a>, <a href="#write_source_files-srcs">srcs</a>, <a href="#write_source_files-strip_prefixes">strip_prefixes</a>, <a href="#write_source_files-target">target</a>)
</pre>



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="write_source_files-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="write_source_files-srcs"></a>srcs |  Input file(s).   | <a href="https://bazel.build/concepts/labels">List of labels</a> | required |  |
| <a id="write_source_files-strip_prefixes"></a>strip_prefixes |  -   | List of strings | optional |  `[]`  |
| <a id="write_source_files-target"></a>target |  -   | String | required |  |


<a id="runfile"></a>

## runfile

<pre>
load("@rules_k8s_cd//lib:utils.bzl", "runfile")

runfile(<a href="#runfile-ctx">ctx</a>, <a href="#runfile-f">f</a>)
</pre>

Return the runfiles relative path of f.

**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="runfile-ctx"></a>ctx |  <p align="center"> - </p>   |  none |
| <a id="runfile-f"></a>f |  <p align="center"> - </p>   |  none |


<a id="download_binary"></a>

## download_binary

<pre>
load("@rules_k8s_cd//lib:utils.bzl", "download_binary")

download_binary(<a href="#download_binary-name">name</a>, <a href="#download_binary-bin">bin</a>, <a href="#download_binary-binaries">binaries</a>)
</pre>

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="download_binary-name"></a>name |  A unique name for this repository.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="download_binary-bin"></a>bin |  -   | String | optional |  `""`  |
| <a id="download_binary-binaries"></a>binaries |  -   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> List of strings</a> | optional |  `{}`  |



#!/bin/bash
set -e

CND=sha256sum
# For mac
if [ ! -x "$(command -v sha256sum)" ]; then
  CMD="shasum -a256"
fi

gh release view ${TAG} --json body -q .body > release.txt
DIGEST=`wget https://github.com/loopingz/rules_k8s_cd/archive/refs/tags/${TAG}.tar.gz -O- | ${CMD} | cut -f1 -d' '`
cat << EOF >> release.txt

## Workspace snippet

\`\`\`starlark
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "com_loopingz_rules_k8s_cd",
    sha256 = "${DIGEST}",
    strip_prefix = "rules_k8s_cd-${TAG}",
    urls = ["https://github.com/loopingz/rules_k8s_cd/archive/refs/tags/${TAG}.tar.gz"],
)
\`\`\`


EOF
gh release edit ${TAG} -F release.txt

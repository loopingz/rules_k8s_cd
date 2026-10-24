#!/bin/bash
# Appends the http_archive snippet to the GitHub release of $RELEASE_TAG.
# Run by the release tool after it tags a release (release_targets).
set -euo pipefail

if [ -z "${RELEASE_TAG:-}" ]; then
  echo "error: RELEASE_TAG is not set (it is set by the release tool)" >&2
  exit 1
fi
TAG="${RELEASE_TAG}"
export GH_TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-${FORGE_TOKEN:-}}}"

CMD=sha256sum
# For mac
if [ ! -x "$(command -v sha256sum)" ]; then
  CMD="shasum -a256"
fi

NOTES="$(mktemp)"
trap 'rm -f "${NOTES}"' EXIT

gh release view "${TAG}" --json body -q .body > "${NOTES}"
# --retry-targets re-runs this script: do not append the snippet twice.
if grep -q '^## Workspace snippet' "${NOTES}"; then
  exit 0
fi
DIGEST=$(wget "https://github.com/loopingz/rules_k8s_cd/archive/refs/tags/${TAG}.tar.gz" -O- | ${CMD} | cut -f1 -d' ')
cat << EOT >> "${NOTES}"

## Workspace snippet

\`\`\`starlark
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "rules_k8s_cd",
    sha256 = "${DIGEST}",
    strip_prefix = "rules_k8s_cd-${TAG:1}",
    urls = ["https://github.com/loopingz/rules_k8s_cd/archive/refs/tags/${TAG}.tar.gz"],
)
\`\`\`


EOT
gh release edit "${TAG}" -F "${NOTES}"

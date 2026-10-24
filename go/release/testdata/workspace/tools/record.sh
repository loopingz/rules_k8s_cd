#!/bin/sh
# Appends "<RELEASE_TAG> <resolved version>" to $IT_RELEASE_LOG and
# "<RELEASE_TAG> <HEAD of $BUILD_WORKSPACE_DIRECTORY>" to $IT_RELEASE_LOG.heads.
# $1 is the runfiles path of a release_version output. `bazel run` starts in
# the runfiles tree: the checkout is only reachable through
# $BUILD_WORKSPACE_DIRECTORY.
set -eu
printf '%s %s\n' "$RELEASE_TAG" "$(cat "$1")" >> "$IT_RELEASE_LOG"
printf '%s %s\n' "$RELEASE_TAG" "$(git -C "${BUILD_WORKSPACE_DIRECTORY:?}" rev-parse HEAD)" >> "$IT_RELEASE_LOG.heads"

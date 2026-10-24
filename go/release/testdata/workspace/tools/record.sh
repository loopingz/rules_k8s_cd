#!/bin/sh
# Appends "<RELEASE_TAG> <resolved version>" to $IT_RELEASE_LOG. $1 is the
# runfiles path of a release_version output.
set -eu
printf '%s %s\n' "$RELEASE_TAG" "$(cat "$1")" >> "$IT_RELEASE_LOG"

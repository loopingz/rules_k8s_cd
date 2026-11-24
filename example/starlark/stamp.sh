#!/bin/bash
# Volatile keys (no STABLE_ prefix): a new commit reruns no stamped action (see lib/stamp.sh).
GIT_COMMIT=$(git rev-parse --short HEAD)
GIT_TAGS=$(git tag)
GIT_PR=$GITHUB_PULLREQUEST

echo "GIT_COMMIT $GIT_COMMIT"
echo "GIT_TAGS $GIT_TAGS"
echo "GIT_PR $GIT_PR"

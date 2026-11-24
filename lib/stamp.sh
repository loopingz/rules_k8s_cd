#!/bin/bash
# Workspace status: keys starting with STABLE_ go to stable-status.txt, and a change of any of them
# reruns every stamped action; the others go to volatile-status.txt, which reruns nothing. So the
# values that change with every commit (or every new tag) are volatile: an action that reruns for
# another reason picks up the current ones. Only the tags at HEAD (a release) are stable.
GIT_COMMIT=$(git rev-parse --short HEAD)
GIT_TAGS=$(git tag)
GIT_PR=$GITHUB_PULLREQUEST
GIT_COMMIT_FULL=$(git rev-parse HEAD)
GIT_COMMIT_DATE=$(git show -s --format=%cI HEAD)
STABLE_GIT_TAGS_AT_HEAD=$(git tag --points-at HEAD | tr '\n' ' ' | sed 's/ *$//')
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "GIT_COMMIT $GIT_COMMIT"
echo "GIT_TAGS $GIT_TAGS"
echo "GIT_PR $GIT_PR"
echo "GIT_COMMIT_FULL $GIT_COMMIT_FULL"
echo "GIT_COMMIT_DATE $GIT_COMMIT_DATE"
echo "STABLE_GIT_TAGS_AT_HEAD $STABLE_GIT_TAGS_AT_HEAD"
echo "GIT_BRANCH $GIT_BRANCH"

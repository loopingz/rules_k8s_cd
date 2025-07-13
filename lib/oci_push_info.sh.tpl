#!/usr/bin/env bash
readonly YQ="{{yq}}"
readonly IMAGE_DIR="{{image_dir}}"
readonly PUSHER="{{pusher}}"

MANIFEST_DIGEST=$(${YQ} -r eval '.manifests[0].digest' "${IMAGE_DIR}/index.json")
echo $MANIFEST_DIGEST > {{digestfile}}

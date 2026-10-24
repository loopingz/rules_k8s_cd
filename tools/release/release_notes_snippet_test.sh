#!/bin/bash
# Tests release_notes_snippet.sh ($1) with fake gh and wget: under `bazel run`
# the working directory is the runfiles tree, so the script must run gh from
# $BUILD_WORKSPACE_DIRECTORY.
set -euo pipefail

SCRIPT="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT
mkdir -p "${TMP}/bin" "${TMP}/ws" "${TMP}/runfiles"
LOG="${TMP}/gh.log"

cat > "${TMP}/bin/gh" << EOT
#!/bin/sh
echo "\$(pwd -P) \$*" >> "${LOG}"
case "\$2" in
  view) echo "## Notes" ;;
  edit) cp "\$5" "${TMP}/notes" ;;
esac
EOT
cat > "${TMP}/bin/wget" << 'EOT'
#!/bin/sh
echo archive
EOT
chmod +x "${TMP}/bin/gh" "${TMP}/bin/wget"
WS="$(cd "${TMP}/ws" && pwd -P)"

fail() { echo "FAIL: $*" >&2; exit 1; }

# Run from the "runfiles tree".
cd "${TMP}/runfiles"
PATH="${TMP}/bin:${PATH}" RELEASE_TAG=v1.2.3 BUILD_WORKSPACE_DIRECTORY="${TMP}/ws" "${SCRIPT}"
[ "$(wc -l < "${LOG}")" -eq 2 ] || fail "gh calls: $(cat "${LOG}")"
while read -r dir _; do
  [ "${dir}" = "${WS}" ] || fail "gh ran in ${dir}, want ${WS}"
done < "${LOG}"
grep -q 'strip_prefix = "rules_k8s_cd-1.2.3"' "${TMP}/notes" || fail "snippet missing: $(cat "${TMP}/notes")"

# Without BUILD_WORKSPACE_DIRECTORY (not run by bazel run) it refuses.
if out="$(PATH="${TMP}/bin:${PATH}" RELEASE_TAG=v1.2.3 BUILD_WORKSPACE_DIRECTORY= "${SCRIPT}" 2>&1)"; then
  fail "ran without BUILD_WORKSPACE_DIRECTORY"
fi
case "${out}" in
  *BUILD_WORKSPACE_DIRECTORY*) ;;
  *) fail "unexpected error: ${out}" ;;
esac
echo PASS

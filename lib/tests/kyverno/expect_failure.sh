#!/usr/bin/env bash
# Runs a kyverno_test script and succeeds only if it fails on a policy violation
# (on exactly $2 failing policies when given).
set -u
out="$("$1" 2>&1)" && { printf '%s\n' "$out"; echo "expected a policy failure" >&2; exit 1; }
printf '%s\n' "$out"
grep -q 'fail: [1-9]' <<<"$out" || { echo "failed, but not on a policy violation" >&2; exit 1; }
if [ -n "${2:-}" ]; then
  grep -q "fail: $2," <<<"$out" || { echo "expected $2 failing policies" >&2; exit 1; }
fi

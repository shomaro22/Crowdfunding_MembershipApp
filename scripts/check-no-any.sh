#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

hits=$(rg '\bany\b' --glob '*.go' --glob '!vendor/**' . || true)
if [[ -z "$hits" ]]; then
  exit 0
fi

echo "error: 'any' type is forbidden (see .cursor/rules/no-any.mdc):" >&2
echo "$hits" >&2
exit 1

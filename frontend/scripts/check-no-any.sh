#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

if rg '\bany\b' --glob '*.{ts,tsx}' src; then
  echo 'error: any keyword found in frontend sources' >&2
  exit 1
fi

echo 'ok: no any in frontend sources'

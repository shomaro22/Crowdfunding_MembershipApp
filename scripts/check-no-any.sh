#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

if rg '\bany\b' --glob '*.go' .; then
	echo 'error: any keyword found in Go sources' >&2
	exit 1
fi

echo 'ok: no any in Go sources'

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# shellcheck source=../_scripts/common.sh
source "${ROOT_DIR}/examples/_scripts/common.sh"

echo "==> Generating casters (pointers example)"
( cd "${ROOT_DIR}" && go run ./cmd/caster-generator gen \
  -pkg ./examples/pointers \
  -mapping ./examples/pointers/map.yaml \
  -out ./examples/pointers/generated \
  -package casters )

echo "==> Compile-check generated package"
( cd "${ROOT_DIR}" && go test ./examples/pointers/generated -run '^$' )

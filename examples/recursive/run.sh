#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EX_DIR="$ROOT_DIR/examples/recursive"

OUT_DIR="$EX_DIR/generated"
mkdir -p "$OUT_DIR"

cd "$ROOT_DIR"

echo "[recursive] gen -> $OUT_DIR"
go run ./cmd/caster-generator gen \
  -pkg ./examples/recursive \
  -mapping "$EX_DIR/map.yaml" \
  -out "$OUT_DIR" \
  -package casters

# compile-check package
go test ./examples/recursive -run '^$'

#!/usr/bin/env bash
set -euo pipefail

# Common helpers for example run.sh scripts.

repo_root() {
  # Resolve repo root from this script location.
  # examples/_scripts/common.sh -> repo root is two levels up.
  local d
  d="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  echo "$d"
}

run_caster_generator() {
  local root
  root="$(repo_root)"

  if [[ "${CG_DEBUG:-}" != "" ]]; then
    echo "+ (cd \"$root\" && go run ./cmd/caster-generator $*)" >&2
  fi

  (cd "$root" && go run ./cmd/caster-generator "$@")
}

clean_dir() {
  local dir="$1"
  rm -rf "$dir"
  mkdir -p "$dir"
}

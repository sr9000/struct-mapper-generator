#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/../_scripts/common.sh"

stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

stage_start "Generate Code" "Generate caster functions for nested collections"

run_gen "${here}/map.yaml" "$out_dir" \
  -pkg ./examples/nested-collections \
  -package generated

test_compile ./examples/nested-collections

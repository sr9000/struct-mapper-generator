#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

out_dir="${here}/generated"
clean_dir "$out_dir"

run_caster_generator gen \
  -pkg ./examples/pointers \
  -mapping "${here}/map.yaml" \
  -out "$out_dir"

# Compile the example package (includes generated code).
root="$(repo_root)"
(cd "$root" && go test ./examples/pointers -run '^$' -count=1)

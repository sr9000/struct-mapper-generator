#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/../_scripts/common.sh"
scenario_init "Virtual With External" "Generate target struct with external types (time.Time)"
clean_dir "${here}/stages"
clean_dir "${here}/generated"
rm -f "${here}/missing_types.go"
# Mapping
mkdir -p "${here}/stages"
cat > "${here}/stages/mapping.yaml" << 'EOF'
version: "1"
mappings:
  - source: caster-generator/examples/virtual-with-external.Event
    target: caster-generator/examples/virtual-with-external.EventDTO
    generate_target: true
    fields:
      - source: ID
        target: ID
      - source: Timestamp
        target: Timestamp
      - source: Data
        target: Data
EOF
run_gen "${here}/stages/mapping.yaml" "${here}/generated"
# Verification
if [[ -f "${here}/missing_types.go" ]]; then
    info "✓ missing_types.go generated"
else
    fail "missing_types.go not found"
fi
# Check import
if grep -q '"time"' "${here}/missing_types.go"; then
    info "✓ time import found"
else
    fail "time import not found in missing_types.go"
fi
if grep -q "Timestamp time.Time" "${here}/missing_types.go"; then
    info "✓ Timestamp field has correct type"
else
    fail "Timestamp field check failed"
fi
stage_start "Compile Check" "Verify generated code compiles"
test_compile "${here}"
if go build -o /dev/null "${here}/generated"/*.go; then
    info "✓ Generated casters compile"
else
    fail "Generated casters failed to compile"
fi
scenario_success

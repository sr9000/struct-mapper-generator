#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/../_scripts/common.sh"
scenario_init "Virtual Pointer Fields" "Generate target struct with pointer fields"
clean_dir "${here}/stages"
clean_dir "${here}/generated"
rm -f "${here}/missing_types.go"
mkdir -p "${here}/stages"
cat > "${here}/stages/mapping.yaml" << 'EOF'
version: "1"
mappings:
  - source: caster-generator/examples/virtual-pointer-fields.Person
    target: caster-generator/examples/virtual-pointer-fields.PersonDTO
    generate_target: true
    fields:
      - source: Name
        target: FullName
      - source: Age
        target: Age
      - source: Address
        target: Location
  - source: caster-generator/examples/virtual-pointer-fields.Address
    target: caster-generator/examples/virtual-pointer-fields.AddressDTO
    generate_target: true
    fields:
      - source: City
        target: CityName
EOF
run_gen "${here}/stages/mapping.yaml" "${here}/generated"
# Verification
if [[ -f "${here}/missing_types.go" ]]; then
    info "✓ missing_types.go generated"
else
    fail "missing_types.go not found"
fi
if grep -q "\*int" "${here}/missing_types.go"; then
    info "✓ Age *int found"
else
    fail "Age *int not found"
fi
if grep -q "\*AddressDTO" "${here}/missing_types.go"; then
    info "✓ Location *AddressDTO found"
else
    fail "Location *AddressDTO not found"
fi
stage_start "Compile Check" "Verify generated code compiles"
test_compile "${here}"
if go build -o /dev/null "${here}/generated"/*.go; then
    info "✓ Generated casters compile"
else
    fail "Generated casters failed to compile"
fi
scenario_success

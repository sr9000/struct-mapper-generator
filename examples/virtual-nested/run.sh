#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/../_scripts/common.sh"
scenario_init "Virtual Nested" "Generate nested target types"
clean_dir "${here}/stages"
clean_dir "${here}/generated"
rm -f "${here}/missing_types.go"
# Mapping
mkdir -p "${here}/stages"
cat > "${here}/stages/mapping.yaml" << 'EOF'
version: "1"
mappings:
  - source: caster-generator/examples/virtual-nested.Order
    target: caster-generator/examples/virtual-nested.OrderDTO
    generate_target: true
    fields:
      - source: ID
        target: ID
      - source: Items
        target: Items
  - source: caster-generator/examples/virtual-nested.OrderItem
    target: caster-generator/examples/virtual-nested.OrderItemDTO
    generate_target: true
    fields:
      - source: SKU
        target: SKU
      - source: Qty
        target: Qty
EOF
run_gen "${here}/stages/mapping.yaml" "${here}/generated"
# Verification
if [[ -f "${here}/missing_types.go" ]]; then
    info "✓ missing_types.go generated"
else
    fail "missing_types.go not found"
fi
if grep -q "type OrderDTO struct" "${here}/missing_types.go"; then
    info "✓ OrderDTO found"
else
    fail "OrderDTO not found"
fi
if grep -q "type OrderItemDTO struct" "${here}/missing_types.go"; then
    info "✓ OrderItemDTO found"
else
    fail "OrderItemDTO not found"
fi
# Verify field reference: Items []OrderItemDTO
if grep -E "Items\s+\[\]OrderItemDTO" "${here}/missing_types.go"; then
    info "✓ Items field references OrderItemDTO correctly"
else
    fail "Items field reference incorrect"
fi
# Compile checks
stage_start "Compile Check" "Verify generated code compiles"
test_compile "${here}"
if go build -o /dev/null "${here}/generated"/*.go; then
    info "✓ Generated casters compile"
else
    fail "Generated casters failed to compile"
fi
scenario_success

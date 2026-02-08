#!/usr/bin/env bash
# Step 11: Collections — Slices, Arrays, and Maps
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 11: Collections — Slices, Arrays, and Maps"

info "caster-generator handles various collection types."
info "Use dive hints for element-wise transformation."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step11")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. Fixed-size arrays with dive"

info "Our types have arrays:"
info "  APIBox.Corners → [4]APIPoint"
info "  DomainBox.Corners → [4]DomainPoint"
echo ""

cat > "${stage_dir}/array_mapping.yaml" << 'EOF'
version: "1"

transforms:
  - name: tutorial.Float64ToInt
    signature: "func(float64) int"

mappings:
  # Box mapping with array dive
  - source: caster-generator/step-by-step/tutorial.APIBox
    target: caster-generator/step-by-step/tutorial.DomainBox
    fields:
      # Array with dive: convert each element
      - source:
          Corners: dive
        target:
          Corners: dive
    "121":
      Name: Label

  # Element mapping: APIPoint → DomainPoint
  - source: caster-generator/step-by-step/tutorial.APIPoint
    target: caster-generator/step-by-step/tutorial.DomainPoint
    fields:
      - source: X
        target: X
        transform: tutorial.Float64ToInt
      - source: Y
        target: Y
        transform: tutorial.Float64ToInt
EOF

show_yaml "${stage_dir}/array_mapping.yaml" "Array mapping with dive:"

prompt_continue

subheader "2. Generate array casters"

run_cg gen \
    -mapping "${stage_dir}/array_mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated array caster:"
    fi
done

info "Notice: Array elements are transformed in a loop."

prompt_continue

subheader "3. Maps with dive"

# Clean for next example
rm -rf "$out_dir"
mkdir -p "$out_dir"

info "Maps work similarly - the key type must be compatible,"
info "and the value type can be converted via dive."
echo ""

cat > "${stage_dir}/map_mapping.yaml" << 'EOF'
version: "1"

transforms:
  - name: tutorial.DollarsToCents
    signature: "func(float64) int64"

mappings:
  # For maps, dive converts each value while preserving keys
  # Note: Map key types must be compatible (string → string)

  # Value mapping: APIProduct → DomainProduct
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial.DomainProduct
    fields:
      - source: PriceUSD
        target: PriceCents
        transform: tutorial.DollarsToCents
    "121":
      ProductID: SKU
      ProductName: Name
      InStock: Stock
      Category: CategoryTag
    ignore:
      - ID
      - CreatedAt
      - UpdatedAt
EOF

show_yaml "${stage_dir}/map_mapping.yaml" "Map value mapping:"

info "Map mapping pattern:"
info "  - Define a mapping for the value type"
info "  - Use dive on the parent map field"
info "  - Keys are preserved, values are transformed"

prompt_continue

subheader "4. Slices (covered in Step 10)"

info "Slices work the same as arrays with dive."
info "See Step 10 for the Items → LineItems example."
info ""
info "Summary of dive for collections:"
info "  • Slices: []T → []U with element caster"
info "  • Arrays: [N]T → [N]U with element caster"
info "  • Maps: map[K]V → map[K]W with value caster (keys unchanged)"

done_step "11"
info "Files are in: ${stage_dir}/"
info "Next: ./step12.sh — Pointers and deep paths"

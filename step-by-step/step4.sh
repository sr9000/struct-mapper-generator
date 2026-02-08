#!/usr/bin/env bash
# Step 4: 121 Mappings — Simple Field Renames
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 4: 121 Mappings — Simple Field Renames"

info "The '121' section provides a compact way to express one-to-one field mappings."
info "Use it when types match and you just need to rename fields."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step4")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. The 121 syntax"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIAddress
    target: caster-generator/step-by-step/tutorial.DomainAddress
    "121":
      # Format: source_field: target_field
      Street: Street       # Same name (explicit)
      City: City           # Same name (explicit)
      ZipCode: PostalCode  # Rename: ZipCode → PostalCode
      Country: Country     # Same name (explicit)
EOF

show_yaml "${stage_dir}/mapping.yaml" "121 mapping syntax:"

info "Key points:"
info "  • Simple key:value pairs"
info "  • source_field: target_field"
info "  • Types must be compatible (same or assignable)"

prompt_continue

subheader "2. Generate code from 121 mapping"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated from 121 mapping:"
    fi
done

prompt_continue

subheader "3. Multiple 121 mappings in one file"

cat > "${stage_dir}/multi_mapping.yaml" << 'EOF'
version: "1"

mappings:
  # First mapping: Address
  - source: caster-generator/step-by-step/tutorial.APIAddress
    target: caster-generator/step-by-step/tutorial.DomainAddress
    "121":
      Street: Street
      City: City
      ZipCode: PostalCode
      Country: Country

  # Second mapping: Point
  - source: caster-generator/step-by-step/tutorial.APIPoint
    target: caster-generator/step-by-step/tutorial.DomainPoint
    "121":
      # Note: These have different types (float64 → int)
      # This won't work without a transform!
      # X: X
      # Y: Y
EOF

show_yaml "${stage_dir}/multi_mapping.yaml" "Multiple mappings in one file:"

info "Notice: APIPoint.X (float64) → DomainPoint.X (int) needs a transform."
info "121 only works for compatible types. We'll cover transforms in Step 7."

done_step "4"
info "Files are in: ${stage_dir}/"
info "Next: ./step5.sh — The fields section for full control"

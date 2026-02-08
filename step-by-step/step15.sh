#!/usr/bin/env bash
# Step 15: Virtual Types — Generate Types On-the-Fly
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 15: Virtual Types — Generate Types On-the-Fly"

info "When target types don't exist, caster-generator can create them."
info "Use 'generate_target: true' to auto-generate struct definitions."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step15")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. The use case"

info "Scenarios where virtual types are useful:"
info "  • Migrating to new domain model"
info "  • Creating DTOs from domain types"
info "  • Prototyping before finalizing types"
info "  • Generating types from external API responses"
echo ""

prompt_continue

subheader "2. Using generate_target"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

mappings:
  # Generate a new 'SimpleProduct' type from APIProduct
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial/generated.SimpleProduct
    generate_target: true    # ← This is the magic flag!
    fields:
      - source: ProductID
        target: ID
      - source: ProductName
        target: Name
      - source: PriceUSD
        target: Price
      - source: InStock
        target: Available
EOF

show_yaml "${stage_dir}/mapping.yaml" "Mapping with generate_target:"

info "Key points:"
info "  • generate_target: true enables type generation"
info "  • Target type will be inferred from field mappings"
info "  • Types are inferred from source field types"

prompt_continue

subheader "3. Generate code with virtual types"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package generated

info "Generated files:"
ls -la "$out_dir"

prompt_continue

subheader "4. Examine generated types"

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated file: $(basename "$f")"
    fi
done

info "Notice:"
info "  • missing_types.go contains the generated struct definition"
info "  • Field types are inferred from source types"
info "  • Caster function references the generated type"

prompt_continue

subheader "5. Explicit type control"

info "You can override inferred types with target_type:"
echo ""

cat > "${stage_dir}/explicit_types.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial/generated.TypedProduct
    generate_target: true
    fields:
      - source: ProductID
        target: ID
        target_type: "int64"    # Override: string → int64

      - source: ProductName
        target: Name

      - source: PriceUSD
        target: PriceCents
        target_type: "int64"    # Override: float64 → int64
EOF

show_yaml "${stage_dir}/explicit_types.yaml" "Explicit type overrides:"

done_step "15"
info "Files are in: ${stage_dir}/"
info "Next: ./step16.sh — Cross-package virtual types"

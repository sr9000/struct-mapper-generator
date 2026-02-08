#!/usr/bin/env bash
# Step 5: Fields Section — Full Control
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 5: Fields Section — Full Control"

info "The 'fields' section gives you complete control over mappings."
info "Use it for transforms, defaults, and complex scenarios."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step5")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. Basic fields syntax"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

transforms:
  - name: tutorial.DollarsToCents
    signature: "func(float64) int64"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial.DomainProduct
    fields:
      # Simple rename (same as 121)
      - source: ProductID
        target: SKU

      # Another rename
      - source: ProductName
        target: Name

      # With transform function
      - source: PriceUSD
        target: PriceCents
        transform: tutorial.DollarsToCents

      # Direct mapping
      - source: InStock
        target: Stock

      # Rename
      - source: Category
        target: CategoryTag
EOF

show_yaml "${stage_dir}/mapping.yaml" "Fields section syntax:"

info "Key points:"
info "  • Each field mapping is a list item"
info "  • 'source' and 'target' specify field names"
info "  • 'transform' specifies a function for type conversion"

prompt_continue

subheader "2. Generate code with fields"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated with transform:"
    fi
done

info "Notice the transform function call: tutorial.DollarsToCents(in.PriceUSD)"

prompt_continue

subheader "3. Mixing 121 and fields"

cat > "${stage_dir}/mixed_mapping.yaml" << 'EOF'
version: "1"

transforms:
  - name: tutorial.DollarsToCents
    signature: "func(float64) int64"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial.DomainProduct

    # 121 for simple renames (higher priority)
    "121":
      ProductID: SKU
      ProductName: Name
      InStock: Stock
      Category: CategoryTag

    # fields for complex mappings
    fields:
      - source: PriceUSD
        target: PriceCents
        transform: tutorial.DollarsToCents
EOF

show_yaml "${stage_dir}/mixed_mapping.yaml" "Mixed 121 and fields:"

info "Priority order: 121 > fields > ignore > auto"
info "Use 121 for simple renames, fields for complex logic."

rm -rf "$out_dir"
mkdir -p "$out_dir"

run_cg gen \
    -mapping "${stage_dir}/mixed_mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated from mixed mapping:"
    fi
done

done_step "5"
info "Files are in: ${stage_dir}/"
info "Next: ./step6.sh — Ignore and auto sections"

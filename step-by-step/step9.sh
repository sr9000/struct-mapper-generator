#!/usr/bin/env bash
# Step 9: Transform Stubs — Auto-Generated Placeholders
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 9: Transform Stubs — Auto-Generated Placeholders"

info "When you reference transforms that don't exist, the generator creates stubs."
info "This helps you get started quickly and fill in implementations later."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step9")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. Using non-existent transforms"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

# These transforms don't exist in the codebase yet!
transforms:
  - name: MyCustomTransform
    signature: "func(string) int64"
  - name: FormatCurrency
    signature: "func(float64, string) string"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial.DomainProduct
    fields:
      # Using a non-existent transform
      - source: ProductID
        target: PriceCents
        transform: MyCustomTransform

    "121":
      ProductName: Name
      InStock: Stock
      Category: CategoryTag

    ignore:
      - ID
      - SKU
      - CreatedAt
      - UpdatedAt
EOF

show_yaml "${stage_dir}/mapping.yaml" "Mapping with non-existent transforms:"

info "MyCustomTransform doesn't exist yet — the generator will create a stub."

prompt_continue

subheader "2. Generate code (creates stubs)"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters || true

info "Generated files:"
ls -la "$out_dir" || true

prompt_continue

subheader "3. Examine generated files"

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated file: $(basename "$f")"
    fi
done

info "Look for missing_transforms.go — it contains stub functions."
info "The stubs panic with 'not implemented' so you know what to fill in."

prompt_continue

subheader "4. Workflow for transform stubs"

info "Recommended workflow:"
info ""
info "  1. Define transforms in YAML (even if they don't exist yet)"
info "  2. Run 'gen' command"
info "  3. Check missing_transforms.go for stubs"
info "  4. Copy stubs to your own file and implement them"
info "  5. Re-run 'gen' — stubs disappear when implementations exist"
info ""

done_step "9"
info "Files are in: ${stage_dir}/"
info "Next: ./step10.sh — Nested structs with dive hints"

#!/usr/bin/env bash
# Step 3: Generate — Creating Caster Code
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 3: Generate — Creating Caster Code"

info "The 'gen' command generates Go code from your mapping YAML."
info "This is where the magic happens!"
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step3")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. Create a simple mapping file"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial.DomainProduct
    "121":
      ProductID: SKU
      ProductName: Name
      InStock: Stock
      Category: CategoryTag
EOF

show_yaml "${stage_dir}/mapping.yaml" "Simple mapping file:"

prompt_continue

subheader "2. Generate caster code"

info "Running: caster-generator gen -mapping mapping.yaml -out ./generated -package casters"
echo ""

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

info "Generated files:"
ls -la "$out_dir"

prompt_continue

subheader "3. Examine generated code"

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated caster function:"
    fi
done

info "Notice:"
info "  • Function name derived from type names"
info "  • Field assignments match our 121 mappings"
info "  • Some fields not mapped (PriceCents, CreatedAt, UpdatedAt)"

done_step "3"
info "Generated code is in: ${out_dir}/"
info "Next: ./step4.sh — Learn about 121 mappings"

#!/usr/bin/env bash
# Step 16: Cross-Package Virtual Types
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 16: Cross-Package Virtual Types"

info "Virtual types can be generated in different packages."
info "Useful for separation of concerns (source pkg → target pkg)."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step16")
out_dir="${stage_dir}/generated"
target_pkg="${stage_dir}/target"
mkdir -p "$out_dir"
mkdir -p "$target_pkg"

subheader "1. Cross-package setup"

info "Scenario:"
info "  • Source: step-by-step/tutorial.APIProduct"
info "  • Target: step-by-step/stages/step16/target.Product (generated)"
info "  • Casters: step-by-step/stages/step16/generated"
echo ""

# Create a minimal target package
cat > "${target_pkg}/doc.go" << 'EOF'
// Package target contains generated domain types.
package target
EOF

prompt_continue

subheader "2. Mapping with cross-package target"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/stages/step16/target.Product
    generate_target: true
    fields:
      - source: ProductID
        target: SKU
      - source: ProductName
        target: Name
      - source: PriceUSD
        target: Price
      - source: InStock
        target: Stock
      - source: Category
        target: Tag
EOF

show_yaml "${stage_dir}/mapping.yaml" "Cross-package virtual type mapping:"

info "Target package: step-by-step/stages/step16/target"
info "The generated type will be placed in that package."

prompt_continue

subheader "3. Generate with multiple packages"

info "Running gen with both packages loaded..."
echo ""

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -pkg "./step-by-step/stages/step16/target" \
    -package casters || warn "Generation may have warnings"

info "Generated files in output directory:"
ls -la "$out_dir" 2>/dev/null || echo "(empty)"

info "Generated files in target package:"
ls -la "$target_pkg" 2>/dev/null || echo "(only doc.go)"

prompt_continue

subheader "4. Examine results"

for f in "$out_dir"/*.go "$target_pkg"/*.go; do
    if [[ -f "$f" && "$(basename "$f")" != "doc.go" ]]; then
        show_go "$f" "Generated: $(basename "$f")"
    fi
done

info "Notice:"
info "  • missing_types.go is generated in the target package"
info "  • Caster functions import the target package"
info "  • Clean separation between source and target"

done_step "16"
info "Files are in: ${stage_dir}/"
info "Next: ./step17.sh — Check command for CI/CD"

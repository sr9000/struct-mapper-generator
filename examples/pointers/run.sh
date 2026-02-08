#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Pointers Example: Pointer deref and deep field paths
# ============================================================================
# Types: pointers.APIOrder -> pointers.DomainOrder
#        pointers.APILineItem -> pointers.DomainLineItem
#
# Goal: Demonstrate pointer deref mapping (*int -> int) and deep source paths
# ============================================================================

scenario_init "Pointers" "Demonstrate pointer dereferencing and deep field path mappings"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

# ============================================================================
# Stage 1: Initial suggest
# ============================================================================
stage_start "Initial Suggest" "Generate base mapping"

run_suggest "${stages_dir}/stage1.yaml" \
  -pkg ./examples/pointers \
  -from "caster-generator/examples/pointers.APIOrder" \
  -to "caster-generator/examples/pointers.DomainOrder"

show_yaml_with_comments "${stages_dir}/stage1.yaml" \
  "Initial mapping - may not handle deep paths automatically"

info "Key observations:"
echo "  • LineItem.Price (*int) -> LineItemPrice (int) uses deep path"
echo "  • The generator handles pointer deref automatically for deep paths"
echo "  • Complex nested mappings (Items, LineItem) are ignored for simplicity"
echo ""

prompt_continue

# ============================================================================
# Stage 2: Patch YAML with explicit deep path mappings
# ============================================================================
stage_start "Patch YAML" "Add deep source path mapping"

cat > "${stages_dir}/stage2.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/examples/pointers.APIOrder
    target: caster-generator/examples/pointers.DomainOrder
    fields:
      # Deep path: extract Price from nested LineItem pointer
      # This demonstrates accessing fields through pointer indirection
      - source: LineItem.Price
        target: LineItemPrice
    auto:
      - source: ID
        target: ID
    ignore:
      # These are complex nested mappings that would require additional
      # casters - we ignore them to focus on the deep path feature
      - Items
      - LineItem
EOF

show_yaml_with_comments "${stages_dir}/stage2.yaml" \
  "Deep path mapping: LineItem.Price -> LineItemPrice"

info "Key feature demonstrated:"
echo "  • Deep source paths like 'LineItem.Price' traverse pointer fields"
echo "  • Generator automatically handles *int -> int deref in deep paths"
echo ""

prompt_continue

# ============================================================================
# Stage 3: Suggest improve
# ============================================================================
stage_start "Suggest Improve" "Validate mapping with suggest"

run_suggest_improve "${stages_dir}/stage2.yaml" "${stages_dir}/stage4.yaml" \
  -pkg ./examples/pointers

show_yaml_with_comments "${stages_dir}/stage4.yaml" \
  "Validated mapping"

prompt_continue

# ============================================================================
# Stage 4: Generate code
# ============================================================================
stage_start "Generate Code" "Generate caster functions"

# Use stage2.yaml which has our explicit mapping
run_gen "${stages_dir}/stage2.yaml" "$out_dir" \
  -pkg ./examples/pointers \
  -package casters

info "Generated files:"
ls -la "$out_dir"

prompt_continue

# ============================================================================
# Stage 5: Compile check
# ============================================================================
stage_start "Compile Check" "Verify generated code compiles"

test_compile ./examples/pointers/generated || {
  warn "Compile check on generated package failed"
  # Try the parent package
  test_compile ./examples/pointers || {
    warn "Compile check failed"
    scenario_partial "Review errors and fix transforms"
    exit 1
  }
}

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping: ${stages_dir}/stage2.yaml"
info "Generated casters: ${out_dir}/"
info ""
info "Key feature demonstrated: Deep source paths (LineItem.Price) with automatic pointer deref"

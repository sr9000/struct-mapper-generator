#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Nested-Mixed Example: Pointers + slices + renames
# ============================================================================
# Types: nestedmixed.APIOrder -> nestedmixed.DomainOrder
#        nestedmixed.APIItem -> nestedmixed.DomainLine
#
# Goal: Demonstrate dive hint for slice of pointers and simple renames
# ============================================================================

scenario_init "Nested-Mixed" "Demonstrate dive hints for pointer slices with renames"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

# ============================================================================
# Stage 1: Initial suggest
# ============================================================================
stage_start "Initial Suggest" "Generate base mapping for order types"

run_suggest "${stages_dir}/stage1.yaml" \
  -pkg ./examples/nested-mixed-structs \
  -from "caster-generator/examples/nested-mixed-structs.APIOrder" \
  -to "caster-generator/examples/nested-mixed-structs.DomainOrder"

show_yaml_with_comments "${stages_dir}/stage1.yaml" \
  "Initial mapping - notice Items -> Lines needs dive hint"

info "Key observations:"
echo "  • Items []*APIItem -> Lines []*DomainLine needs dive"
echo "  • Quantity -> Qty is a rename"
echo "  • Note -> NoteText is a rename (both are *string)"
echo ""

prompt_continue

# ============================================================================
# Stage 2: Patch YAML with dive hint
# ============================================================================
stage_start "Patch YAML" "Add dive hint for Items -> Lines slice mapping"

cat > "${stages_dir}/stage2.yaml" << 'EOF'
version: "1"

# Using 'mappings' format
mappings:
  - source: caster-generator/examples/nested-mixed-structs.APIOrder
    target: caster-generator/examples/nested-mixed-structs.DomainOrder
    fields:
      - source: ID
        target: ID
      - source: Items
        target: Lines
        hint: dive  # Enable element-wise conversion for []*APIItem -> []*DomainLine

  - source: caster-generator/examples/nested-mixed-structs.APIItem
    target: caster-generator/examples/nested-mixed-structs.DomainLine
    fields:
      - source: SKU
        target: SKU
      - source: Quantity
        target: Qty       # Rename
      - source: Note
        target: NoteText  # Rename (both *string)
EOF

show_yaml_with_comments "${stages_dir}/stage2.yaml" \
  "Added dive hint and explicit element type mapping with renames"

prompt_continue

# ============================================================================
# Stage 3: Suggest improve
# ============================================================================
stage_start "Suggest Improve" "Validate mapping with suggest"

run_suggest_improve "${stages_dir}/stage2.yaml" "${stages_dir}/stage3.yaml" \
  -pkg ./examples/nested-mixed-structs

show_yaml_with_comments "${stages_dir}/stage3.yaml" \
  "Validated mapping"

prompt_continue

# ============================================================================
# Stage 4: Generate code
# ============================================================================
stage_start "Generate Code" "Generate caster functions"

# Use stage2.yaml which has the dive hint preserved
run_gen "${stages_dir}/stage2.yaml" "$out_dir" \
  -pkg ./examples/nested-mixed-structs \
  -package casters

info "Generated files:"
ls -la "$out_dir"

prompt_continue

# ============================================================================
# Stage 5: Compile check
# ============================================================================
stage_start "Compile Check" "Verify generated code compiles"

test_compile ./examples/nested-mixed-structs

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping: ${stages_dir}/stage3.yaml"
info "Generated casters: ${out_dir}/"

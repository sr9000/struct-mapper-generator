#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Virtual Example: Generate Target Struct On-the-fly
# ============================================================================
# Types: virtual.Source -> virtual.Target (generated)
#
# Goal: Demonstrate auto-generation of target struct type when no target
#       type exists. The caster-generator will infer the target type shape
#       from the source and mapping configuration.
# ============================================================================

scenario_init "Virtual" "Demonstrate target struct generation from source type"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

# ============================================================================
# Stage 1: Show source type structure
# ============================================================================
stage_start "Examine Source Type" "Review the source type that will be mapped"

show_file "${here}/source.go" "Source types (virtual.Source and virtual.SourceItem)"

info "We will generate a Target struct automatically from the Source struct."
info "The target type does not exist - it will be created by the generator."

prompt_continue

# ============================================================================
# Stage 2: Create mapping file with target type generation
# ============================================================================
stage_start "Define Mapping with Generated Target" "Create mapping with generate_target: true"

cat > "${stages_dir}/stage1_mapping.yaml" << 'EOF'
version: "1"

mappings:
  # Source -> Target (generated type)
  - source: caster-generator/examples/virtual.Source
    target: caster-generator/examples/virtual.Target
    generate_target: true
    fields:
      # Direct mappings
      - source: ID
        target: ID
      - source: Name
        target: Name
      - source: ExtraInfo
        target: Description
    auto:
      - source: Items
        target: Items

  # SourceItem -> TargetItem (generated nested type)
  - source: caster-generator/examples/virtual.SourceItem
    target: caster-generator/examples/virtual.TargetItem
    generate_target: true
    fields:
      - source: ProductID
        target: ProductID
      - source: Quantity
        target: Qty
EOF

show_yaml_with_comments "${stages_dir}/stage1_mapping.yaml" \
  "Mapping file with generate_target: true for virtual target types"

prompt_continue

# ============================================================================
# Stage 3: Generate code with target struct
# ============================================================================
stage_start "Generate Code" "Generate caster functions and target struct definitions"

run_gen "${stages_dir}/stage1_mapping.yaml" "$out_dir" \
  -pkg ./examples/virtual \
  -package casters

info "Generated files:"
ls -la "$out_dir"

# Show generated target struct if exists
for f in "$out_dir"/*.go; do
  if [[ -f "$f" ]]; then
    echo ""
    info "Contents of $(basename "$f"):"
    head -80 "$f"
    echo "..."
  fi
done

prompt_continue

# ============================================================================
# Stage 4: Verify generated struct contains expected fields
# ============================================================================
stage_start "Verify Generated Struct" "Check generated target type has correct fields"

verify_ok=true

# Check for Target struct
if grep -q "type Target struct" "$out_dir"/*.go 2>/dev/null; then
  info "✓ Target struct generated"
else
  warn "✗ Target struct not found"
  verify_ok=false
fi

# Check for expected fields in Target
for field in "ID" "Name" "Description" "Items"; do
  if grep -E "^\s+${field}\s+" "$out_dir"/*.go 2>/dev/null | head -1; then
    info "✓ Field ${field} found in generated struct"
  else
    warn "✗ Field ${field} not found"
    verify_ok=false
  fi
done

# Check for TargetItem struct
if grep -q "type TargetItem struct" "$out_dir"/*.go 2>/dev/null; then
  info "✓ TargetItem struct generated"
else
  warn "✗ TargetItem struct not found"
  verify_ok=false
fi

if [[ "$verify_ok" == "true" ]]; then
  stage_done "All expected fields verified"
else
  warn "Some verifications failed"
fi

prompt_continue

# ============================================================================
# Stage 5: Compile check
# ============================================================================
stage_start "Compile Check" "Verify generated code compiles"

test_compile ./examples/virtual

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping: ${stages_dir}/stage1_mapping.yaml"
info "Generated casters: ${out_dir}/"
info ""
info "The generated Target struct should contain:"
info "  - ID string"
info "  - Name *string"
info "  - Items []*TargetItem"
info "  - Description *string"

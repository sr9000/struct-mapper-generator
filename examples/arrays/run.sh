#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Arrays Example: Dive hints for fixed-size arrays
# ============================================================================
# Types: arrays.APIBox -> arrays.DomainBox
#        arrays.APIPoint -> arrays.DomainPoint
#
# Goal: Demonstrate array element mapping that needs `hint: dive`
# ============================================================================

scenario_init "Arrays" "Demonstrate dive hints for fixed-size array element mapping"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

# ============================================================================
# Stage 1: Initial suggest (no hints)
# ============================================================================
stage_start "Initial Suggest" "Generate base mapping without any hints"

run_suggest "${stages_dir}/stage1.yaml" \
  -pkg ./examples/arrays \
  -from "caster-generator/examples/arrays.APIBox" \
  -to "caster-generator/examples/arrays.DomainBox"

show_yaml_with_comments "${stages_dir}/stage1.yaml" \
  "Notice: Corners field is detected but may not map element types automatically"

prompt_continue

# ============================================================================
# Stage 2: Patch YAML - add dive hint
# ============================================================================
stage_start "Patch YAML" "Add hint: dive for array field to enable element conversion"

# Create a proper mapping with dive hint
cat > "${stages_dir}/stage2.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/examples/arrays.APIBox
    target: caster-generator/examples/arrays.DomainBox
    fields:
      - source: Corners
        target: Corners
        hint: dive  # Enable element-wise conversion for the array

  - source: caster-generator/examples/arrays.APIPoint
    target: caster-generator/examples/arrays.DomainPoint
    121:
      X: X
      Y: Y
EOF

show_yaml_with_comments "${stages_dir}/stage2.yaml" \
  "Added hint: dive to Corners field + explicit APIPoint -> DomainPoint mapping"

info "Key changes:"
echo "  • Added 'hint: dive' to Corners field mapping"
echo "  • Added explicit mapping for APIPoint -> DomainPoint"
echo ""

prompt_continue

# ============================================================================
# Stage 3: Suggest with existing mapping (improve)
# ============================================================================
stage_start "Suggest Improve" "Run suggest with patched mapping to verify/improve"

run_suggest_improve "${stages_dir}/stage2.yaml" "${stages_dir}/stage3.yaml" \
  -pkg ./examples/arrays

show_yaml_with_comments "${stages_dir}/stage3.yaml" \
  "Improved mapping with suggestions filled in"

prompt_continue

# ============================================================================
# Stage 4: Generate code
# ============================================================================
stage_start "Generate Code" "Generate caster functions from final mapping"

run_gen "${stages_dir}/stage3.yaml" "$out_dir" \
  -pkg ./examples/arrays \
  -package casters

info "Generated files:"
ls -la "$out_dir"

prompt_continue

# ============================================================================
# Stage 5: Compile check
# ============================================================================
stage_start "Compile Check" "Verify generated code compiles"

test_compile ./examples/arrays

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping is in: ${stages_dir}/stage3.yaml"
info "Generated code is in: ${out_dir}/"

#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Recursive Example: Self-referential types
# ============================================================================
# Types: recursive.Node -> recursive.NodeDTO
#
# Goal: Demonstrate recursive caster generation (linked list pattern)
# ============================================================================

scenario_init "Recursive" "Demonstrate recursive type mapping (linked list)"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

# ============================================================================
# Stage 1: Analyze types
# ============================================================================
stage_start "Analyze Types" "Examine the recursive structure"

info "Source type: recursive.Node"
echo "  Value int"
echo "  Next  *Node  <- self-reference"
echo ""
info "Target type: recursive.NodeDTO"
echo "  Value int"
echo "  Next  *NodeDTO  <- self-reference"
echo ""

prompt_continue

# ============================================================================
# Stage 2: Initial suggest
# ============================================================================
stage_start "Initial Suggest" "Generate mapping for Node -> NodeDTO"

run_suggest "${stages_dir}/stage2.yaml" \
  -pkg ./examples/recursive \
  -from "caster-generator/examples/recursive.Node" \
  -to "caster-generator/examples/recursive.NodeDTO"

show_yaml_with_comments "${stages_dir}/stage2.yaml" \
  "Mapping for recursive type - Next field should map to itself"

info "Key observation:"
echo "  • Next (*Node) -> Next (*NodeDTO) creates recursive caster call"
echo ""

prompt_continue

# ============================================================================
# Stage 3: Verify/patch mapping (usually straightforward for 1:1 recursive)
# ============================================================================
stage_start "Verify Mapping" "Ensure mapping handles recursion correctly"

cat > "${stages_dir}/stage3.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/examples/recursive.Node
    target: caster-generator/examples/recursive.NodeDTO
    121:
      Value: Value
      Next: Next    # Recursive: *Node -> *NodeDTO
EOF

show_yaml_with_comments "${stages_dir}/stage3.yaml" \
  "Clean 1:1 mapping - recursive call handled automatically"

prompt_continue

# ============================================================================
# Stage 4: Suggest improve
# ============================================================================
stage_start "Suggest Improve" "Validate mapping"

run_suggest_improve "${stages_dir}/stage3.yaml" "${stages_dir}/stage4.yaml" \
  -pkg ./examples/recursive

show_yaml_with_comments "${stages_dir}/stage4.yaml" \
  "Validated mapping"

prompt_continue

# ============================================================================
# Stage 5: Generate code
# ============================================================================
stage_start "Generate Code" "Generate recursive caster"

run_gen "${stages_dir}/stage4.yaml" "$out_dir" \
  -pkg ./examples/recursive \
  -package casters

info "Generated files:"
ls -la "$out_dir"

# Show the generated code (recursive caster is interesting)
if [[ -f "$out_dir"/*.go ]]; then
  for f in "$out_dir"/*.go; do
    show_file "$f" "Generated: $(basename "$f")"
  done
fi

prompt_continue

# ============================================================================
# Stage 6: Compile check
# ============================================================================
stage_start "Compile Check" "Verify recursive caster compiles"

test_compile ./examples/recursive

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping: ${stages_dir}/stage4.yaml"
info "Generated casters: ${out_dir}/"
info ""
info "The generated caster handles recursion by calling itself for the Next field."

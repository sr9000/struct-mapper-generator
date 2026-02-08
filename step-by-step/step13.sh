#!/usr/bin/env bash
# Step 13: Recursive Types — Self-Referential Structs
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 13: Recursive Types — Self-Referential Structs"

info "caster-generator handles recursive/self-referential types safely."
info "Common patterns: linked lists, trees, graphs."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step13")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. Understanding recursive types"

info "Our types have self-references:"
info "  TreeNode.Children → []*TreeNode"
info "  TreeNodeDTO.Children → []*TreeNodeDTO"
echo ""
info "The caster needs to call itself recursively."

prompt_continue

subheader "2. Mapping recursive types"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/step-by-step/tutorial.TreeNode
    target: caster-generator/step-by-step/tutorial.TreeNodeDTO
    fields:
      # Children is self-referential: []*TreeNode → []*TreeNodeDTO
      # Use dive to recursively apply this same mapping
      - source:
          Children: dive
        target:
          Children: dive
    "121":
      Value: Data
EOF

show_yaml "${stage_dir}/mapping.yaml" "Recursive type mapping:"

info "Key insight: dive on a self-referential field creates recursive caster calls."

prompt_continue

subheader "3. Generate recursive caster"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated recursive caster:"
    fi
done

info "Notice:"
info "  • The function calls itself for each child"
info "  • Nil checks prevent infinite recursion on terminal nodes"
info "  • Slice iteration handles multiple children"

prompt_continue

subheader "4. How recursion works"

info "Generated pattern:"
info ""
info "  func TreeNodeToTreeNodeDTO(in TreeNode) TreeNodeDTO {"
info "      out := TreeNodeDTO{"
info "          Data: in.Value,"
info "      }"
info "      for _, child := range in.Children {"
info "          if child != nil {"
info "              converted := TreeNodeToTreeNodeDTO(*child)  // ← recursive!"
info "              out.Children = append(out.Children, &converted)"
info "          }"
info "      }"
info "      return out"
info "  }"
info ""
info "The recursion naturally terminates when Children is empty or nil."

done_step "13"
info "Files are in: ${stage_dir}/"
info "Next: ./step14.sh — Context passing with requires and extra"

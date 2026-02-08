#!/usr/bin/env bash
# Step 7: Transforms Basics — Handling Type Mismatches
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 7: Transforms Basics — Handling Type Mismatches"

info "Transform functions convert between incompatible types."
info "Use them when source and target field types differ."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step7")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. When you need transforms"

info "Common scenarios:"
info "  • float64 (dollars) → int64 (cents)"
info "  • string (ID) → uint (database ID)"
info "  • string ('Y'/'N') → bool"
info "  • string (date) → time.Time"
echo ""

prompt_continue

subheader "2. Declaring transforms in YAML"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

# Declare transform functions with their signatures
transforms:
  - name: tutorial.StringToUint
    signature: "func(string) uint"
  - name: tutorial.DollarsToCents
    signature: "func(float64) int64"
  - name: tutorial.YNToBool
    signature: "func(string) bool"
  - name: tutorial.ConcatNames
    signature: "func(string, string) string"

mappings:
  - source: caster-generator/step-by-step/tutorial.APICustomer
    target: caster-generator/step-by-step/tutorial.DomainCustomer
    fields:
      # string → uint
      - source: CustomerID
        target: ID
        transform: tutorial.StringToUint

      # Multiple source fields → single target (N:1)
      - source: [FirstName, LastName]
        target: FullName
        transform: tutorial.ConcatNames

      # string "Y"/"N" → bool
      - source: Active
        target: IsActive
        transform: tutorial.YNToBool

    "121":
      Email: Email
      Phone: Phone

    ignore:
      - CreatedAt
      - UpdatedAt
EOF

show_yaml "${stage_dir}/mapping.yaml" "Mapping with transforms:"

info "Key points:"
info "  • transforms section declares available functions"
info "  • fields.transform references the function name"
info "  • Multi-source (N:1) uses array syntax: [Field1, Field2]"

prompt_continue

subheader "3. Generate code with transforms"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated caster with transforms:"
    fi
done

info "Notice:"
info "  • Transform function calls: tutorial.StringToUint(in.CustomerID)"
info "  • Multi-source: tutorial.ConcatNames(in.FirstName, in.LastName)"

prompt_continue

subheader "4. The transform implementations"

show_file "${here}/tutorial/transforms.go" "Transform functions (already implemented):"

info "These transforms are already in step-by-step/tutorial/transforms.go"
info "The generated code will call these functions."

prompt_continue

subheader "5. Handling Missing Transforms (TODOs)"

info "What if you map incompatible fields but don't provide a transform?"
info "Use 'caster-generator suggest' to generate placeholders."
echo ""

# Create a mapping with type mismatch (String -> Uint)
cat > "${stage_dir}/product_map.yaml" << 'EOF'
version: "1"
mappings:
  - source: caster-generator/step-by-step/tutorial.APIProduct
    target: caster-generator/step-by-step/tutorial.DomainProduct
    fields:
      - source: ProductID   # string
        target: ID          # uint
        # Missing transform!
    ignore:
      - Name
      - PriceCents
      - Stock
      - CategoryTag
      - CreatedAt
      - UpdatedAt
      - SKU
EOF

show_yaml "${stage_dir}/product_map.yaml" "Mapping with missing transform:"

info "Running 'suggest' command..."
run_cg suggest \
    -mapping "${stage_dir}/product_map.yaml" \
    -out "${stage_dir}/product_map_suggested.yaml" \
    -pkg ./step-by-step/tutorial

show_yaml "${stage_dir}/product_map_suggested.yaml" "Suggested mapping (autoplaced TODO):"

info "Now generating code from the suggestion..."
run_cg gen \
    -mapping "${stage_dir}/product_map_suggested.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

if [[ -f "$out_dir/missing_transforms.go" ]]; then
    show_go "$out_dir/missing_transforms.go" "Generated missing_transforms.go:"
fi

info "Notice:"
info "  • The generator created a placeholder stub that panics."
info "  • You can now implement this function in your own file."

done_step "7"
info "Files are in: ${stage_dir}/"
info "Next: ./step8.sh — Multi-source transforms (N:1)"

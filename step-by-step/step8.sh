#!/usr/bin/env bash
# Step 8: Multi-Source Transforms — Combining Fields (N:1)
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 8: Multi-Source Transforms — Combining Fields (N:1)"

info "Sometimes you need to combine multiple source fields into one target field."
info "This is the N:1 pattern, enabled by multi-source transforms."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step8")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. The N:1 pattern"

info "Common scenarios:"
info "  • FirstName + LastName → FullName"
info "  • Street + City + Zip → Address"
info "  • Day + Month + Year → Date"
echo ""

prompt_continue

subheader "2. YAML syntax for multi-source"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

transforms:
  # Multi-argument transform: takes 2 strings, returns 1 string
  - name: tutorial.ConcatNames
    signature: "func(string, string) string"

mappings:
  - source: caster-generator/step-by-step/tutorial.APICustomer
    target: caster-generator/step-by-step/tutorial.DomainCustomer
    fields:
      # N:1 mapping: combine FirstName and LastName into FullName
      # Use array syntax for multiple source fields
      - source: [FirstName, LastName]
        target: FullName
        transform: tutorial.ConcatNames

    "121":
      Email: Email
      Phone: Phone

    ignore:
      - ID
      - IsActive
      - CreatedAt
      - UpdatedAt
EOF

show_yaml "${stage_dir}/mapping.yaml" "Multi-source mapping:"

info "Key syntax: source: [Field1, Field2]"
info "Transform must accept matching number of arguments."

prompt_continue

subheader "3. The transform function"

cat << 'EOF'
// ConcatNames combines first and last name.
func ConcatNames(first, last string) string {
    return strings.TrimSpace(first + " " + last)
}
EOF

echo ""
info "The function signature matches the source fields:"
info "  [FirstName, LastName] → func(first, last string)"

prompt_continue

subheader "4. Generate and examine code"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated N:1 transform call:"
    fi
done

info "Notice: tutorial.ConcatNames(in.FirstName, in.LastName)"
info "Both source fields are passed as arguments."

done_step "8"
info "Files are in: ${stage_dir}/"
info "Next: ./step9.sh — Auto-generated transform stubs"

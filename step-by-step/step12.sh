#!/usr/bin/env bash
# Step 12: Pointers & Deep Paths
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 12: Pointers & Deep Paths"

info "caster-generator automatically handles pointer types."
info "Deep paths let you access fields through nested structures."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step12")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. Automatic pointer handling"

info "The generator handles these conversions automatically:"
info "  • *T → T  (dereference with nil check)"
info "  • T → *T  (wrap in pointer)"
info "  • *T → *T (copy or deep copy)"
echo ""

cat > "${stage_dir}/pointer_mapping.yaml" << 'EOF'
version: "1"

transforms:
  - name: tutorial.IntPtrToInt
    signature: "func(*int) int"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIOrder
    target: caster-generator/step-by-step/tutorial.DomainOrder
    fields:
      # Pointer dereference: *int → int
      - source: ShippingQty
        target: ShippingCount
        transform: tutorial.IntPtrToInt

    "121":
      OrderID: ExternalRef
      Status: OrderStatus

    ignore:
      - ID
      - CustomerID
      - Customer
      - LineItems
      - TotalCents
      - PlacedAt
      - CreatedAt
      - UpdatedAt
EOF

show_yaml "${stage_dir}/pointer_mapping.yaml" "Pointer handling example:"

prompt_continue

subheader "2. Generate with pointer handling"

run_cg gen \
    -mapping "${stage_dir}/pointer_mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated pointer handling:"
    fi
done

info "The transform handles nil checking for *int → int."

prompt_continue

subheader "3. Deep source paths"

# Clean for next example
rm -rf "$out_dir"
mkdir -p "$out_dir"

info "Deep paths access nested fields: Parent.Child.Field"
info "Useful for flattening nested structures."
echo ""

cat > "${stage_dir}/deep_path_mapping.yaml" << 'EOF'
version: "1"

mappings:
  # Example: Extract Customer.Email directly to a flattened field
  # Note: This is a simplified example showing the syntax
  - source: caster-generator/step-by-step/tutorial.APIOrder
    target: caster-generator/step-by-step/tutorial.DomainOrder
    fields:
      # Deep path through pointer: Customer.Email
      # This accesses in.Customer.Email (through *APICustomer)
      # Note: Requires Customer to be non-nil

    "121":
      OrderID: ExternalRef
      Status: OrderStatus

    ignore:
      - ID
      - CustomerID
      - Customer
      - Items
      - LineItems
      - TotalCents
      - PlacedAt
      - ShippingCount
      - CreatedAt
      - UpdatedAt
EOF

show_yaml "${stage_dir}/deep_path_mapping.yaml" "Deep path syntax:"

info "Deep path syntax: source: Parent.Child.Field"
info "The generator adds nil checks for pointer paths."

prompt_continue

subheader "4. Summary of pointer/path features"

info "Pointer handling:"
info "  • Automatic nil checks for source pointers"
info "  • Automatic wrapping for target pointers"
info "  • Works with transforms"
echo ""
info "Deep paths:"
info "  • Dot notation: Field.SubField.SubSubField"
info "  • Works through pointer fields"
info "  • Useful for flattening nested structures"

done_step "12"
info "Files are in: ${stage_dir}/"
info "Next: ./step13.sh — Recursive types"

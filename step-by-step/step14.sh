#!/usr/bin/env bash
# Step 14: Context Passing — Requires & Extra
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 14: Context Passing — Requires & Extra"

info "Sometimes child casters need data from their parent context."
info "Use 'requires' and 'extra' to pass values between casters."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step14")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. The problem"

info "When mapping Order.Items → Order.LineItems:"
info "  • Each DomainLineItem needs an OrderID"
info "  • But APIOrderItem doesn't have an OrderID"
info "  • The OrderID comes from the parent Order"
echo ""
info "Solution: Pass OrderID from parent to child caster."

prompt_continue

subheader "2. Using requires and extra"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

transforms:
  - name: tutorial.StringToUint
    signature: "func(string) uint"
  - name: tutorial.DollarsToCents
    signature: "func(float64) int64"
  - name: tutorial.PassOrderID
    signature: "func(uint) uint"

mappings:
  # Parent mapping: APIOrder → DomainOrder
  - source: caster-generator/step-by-step/tutorial.APIOrder
    target: caster-generator/step-by-step/tutorial.DomainOrder
    fields:
      # First, map OrderID so it's available on 'out'
      - source: OrderID
        target: ID
        transform: tutorial.StringToUint

      - source: OrderID
        target: ExternalRef

      # Items → LineItems with context passing
      - source:
          Items: dive
        target:
          LineItems: dive
        extra:
          # Pass OrderID to child caster
          - name: OrderID
            def:
              target: ID    # Use out.ID (already assigned above)

      - source: TotalUSD
        target: TotalCents
        transform: tutorial.DollarsToCents

    "121":
      Status: OrderStatus

    ignore:
      - CustomerID
      - Customer
      - PlacedAt
      - ShippingCount
      - CreatedAt
      - UpdatedAt

  # Child mapping: APIOrderItem → DomainLineItem
  - source: caster-generator/step-by-step/tutorial.APIOrderItem
    target: caster-generator/step-by-step/tutorial.DomainLineItem

    # REQUIRES: This caster expects OrderID to be passed in
    requires:
      - name: OrderID
        type: uint

    fields:
      # Use the passed-in OrderID with a transform
      # The transform receives the requires parameter
      - source: OrderID
        target: OrderID
        transform: tutorial.PassOrderID

      - source: ProductID
        target: ProductSKU

      - source: UnitPrice
        target: PriceCents
        transform: tutorial.DollarsToCents

    "121":
      Name: ProductName
      Quantity: Qty

    ignore:
      - ID
EOF

show_yaml "${stage_dir}/mapping.yaml" "Context passing with requires/extra:"

info "Key concepts:"
info "  • requires: Declares what the child caster needs"
info "  • extra: Specifies what the parent passes"
info "  • def.target: Reference already-assigned output field"
info "  • def.requires: Use the requires parameter value"

prompt_continue

subheader "3. Generate with context passing"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated casters with context:"
    fi
done

info "Notice:"
info "  • Child caster has extra parameter: func(..., OrderID uint)"
info "  • Parent passes out.ID when calling child caster"

prompt_continue

subheader "4. Extra sources"

info "'extra' can reference different sources:"
info ""
info "  extra:"
info "    - name: OrderID"
info "      def:"
info "        source: SomeField    # From input struct (in.SomeField)"
info ""
info "  extra:"
info "    - name: OrderID"
info "      def:"
info "        target: ID           # From output struct (out.ID)"
info ""
info "  extra:"
info "    - name: OrderID"
info "      def:"
info "        requires: ParentOrderID  # From parent's requires"

done_step "14"
info "Files are in: ${stage_dir}/"
info "Next: ./step15.sh — Virtual types (generate_target)"

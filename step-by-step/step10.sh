#!/usr/bin/env bash
# Step 10: Nested Structs — The Dive Hint
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 10: Nested Structs — The Dive Hint"

info "When you have structs inside structs, use 'dive' to map nested types."
info "This tells the generator to look for a mapping for the inner type."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step10")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "1. Understanding nested types"

info "Our types have nested structures:"
info "  APIOrder.Customer → *APICustomer"
info "  APIOrder.Items → []APIOrderItem"
info ""
info "Target types have different nested structures:"
info "  DomainOrder.Customer → *DomainCustomer"
info "  DomainOrder.LineItems → []DomainLineItem"
echo ""

prompt_continue

subheader "2. Using dive hint for nested structs"

cat > "${stage_dir}/mapping.yaml" << 'EOF'
version: "1"

transforms:
  - name: tutorial.StringToUint
    signature: "func(string) uint"
  - name: tutorial.DollarsToCents
    signature: "func(float64) int64"
  - name: tutorial.ParseISODate
    signature: "func(string) time.Time"
  - name: tutorial.ConcatNames
    signature: "func(string, string) string"
  - name: tutorial.YNToBool
    signature: "func(string) bool"
  - name: tutorial.IntPtrToInt
    signature: "func(*int) int"

mappings:
  # Root mapping: APIOrder → DomainOrder
  - source: caster-generator/step-by-step/tutorial.APIOrder
    target: caster-generator/step-by-step/tutorial.DomainOrder
    fields:
      - source: OrderID
        target: ExternalRef

      - source: CustomerID
        target: CustomerID
        transform: tutorial.StringToUint

      # DIVE: nested struct - use mapping for APICustomer → DomainCustomer
      - source:
          Customer: dive
        target:
          Customer: dive

      # DIVE: slice of structs - use mapping for APIOrderItem → DomainLineItem
      - source:
          Items: dive
        target:
          LineItems: dive

      - source: TotalUSD
        target: TotalCents
        transform: tutorial.DollarsToCents

      - source: OrderDate
        target: PlacedAt
        transform: tutorial.ParseISODate

      - source: ShippingQty
        target: ShippingCount
        transform: tutorial.IntPtrToInt

    "121":
      Status: OrderStatus

    ignore:
      - ID
      - CreatedAt
      - UpdatedAt

  # Nested mapping: APICustomer → DomainCustomer
  - source: caster-generator/step-by-step/tutorial.APICustomer
    target: caster-generator/step-by-step/tutorial.DomainCustomer
    fields:
      - source: CustomerID
        target: ID
        transform: tutorial.StringToUint

      - source: [FirstName, LastName]
        target: FullName
        transform: tutorial.ConcatNames

      - source: Active
        target: IsActive
        transform: tutorial.YNToBool

    "121":
      Email: Email
      Phone: Phone

    ignore:
      - CreatedAt
      - UpdatedAt

  # Nested mapping: APIOrderItem → DomainLineItem
  - source: caster-generator/step-by-step/tutorial.APIOrderItem
    target: caster-generator/step-by-step/tutorial.DomainLineItem
    fields:
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
      - OrderID  # We'll handle this with context passing in Step 14
EOF

show_yaml "${stage_dir}/mapping.yaml" "Mapping with dive hints:"

info "Key syntax for dive:"
info "  source:"
info "    FieldName: dive"
info "  target:"
info "    FieldName: dive"

prompt_continue

subheader "3. Generate nested casters"

run_cg gen \
    -mapping "${stage_dir}/mapping.yaml" \
    -out "$out_dir" \
    -pkg ./step-by-step/tutorial \
    -package casters

info "Generated files:"
ls -la "$out_dir"

prompt_continue

subheader "4. Examine generated code"

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated nested casters:"
    fi
done

info "Notice:"
info "  • Multiple caster functions generated"
info "  • Parent caster calls child casters for nested types"
info "  • Slice dive generates loop with child caster calls"

done_step "10"
info "Files are in: ${stage_dir}/"
info "Next: ./step11.sh — Collections (slices, arrays, maps)"

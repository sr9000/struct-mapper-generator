#!/usr/bin/env bash
# Step 18: Full Workflow — Real-World Scenario
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 18: Full Workflow — Real-World Scenario"

info "Let's put everything together in a complete workflow."
info "We'll map store.Order → warehouse.Order with all features."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step18")
out_dir="${stage_dir}/generated"
mkdir -p "$out_dir"

subheader "Phase 1: Analyze Types"

info "First, explore the available types."
echo ""

run_cg analyze -pkg ./store -pkg ./warehouse -type Order

prompt_continue

subheader "Phase 2: Generate Initial Suggestions"

info "Get smart suggestions for the mapping."
echo ""

run_cg suggest \
    -pkg ./store \
    -pkg ./warehouse \
    -from "caster-generator/store.Order" \
    -to "caster-generator/warehouse.Order" \
    -min-confidence 0.5 \
    -out "${stage_dir}/phase2_suggested.yaml"

show_yaml "${stage_dir}/phase2_suggested.yaml" "Initial suggestions:"

prompt_continue

subheader "Phase 3: Review and Patch"

info "Now we patch the YAML with our knowledge:"
info "  • Add transforms for type conversions"
info "  • Add dive hints for nested types"
info "  • Add ignores for ORM fields"
info "  • Add requires/extra for context passing"
echo ""

cat > "${stage_dir}/phase3_patched.yaml" << 'EOF'
version: "1"

# Declare all transform functions
transforms:
  - name: store_transforms.Int64ToUint
    signature: "func(int64) uint"
  - name: store_transforms.OrderStatusToString
    signature: "func(store.OrderStatus) string"
  - name: store_transforms.PassOrderID
    signature: "func(uint) uint"

mappings:
  # Main mapping: store.Order → warehouse.Order
  - source: caster-generator/store.Order
    target: caster-generator/warehouse.Order
    fields:
      # ID conversion: int64 → uint
      - source: ID
        target: ID
        transform: store_transforms.Int64ToUint

      # CustomerID conversion
      - source: CustomerID
        target: CustomerID
        transform: store_transforms.Int64ToUint

      # Status enum to string
      - source: Status
        target: Status
        transform: store_transforms.OrderStatusToString

      # TotalCents → TotalAmount (same type, rename)
      - source: TotalCents
        target: TotalAmount

      # Items with dive and context passing
      - source:
          Items: dive
        target:
          Items: dive
        extra:
          - name: OrderID
            def:
              target: ID  # Pass out.ID to child caster

      # OrderedAt → PlacedAt (rename, pointer)
      - source: OrderedAt
        target: PlacedAt

    ignore:
      # ORM/database fields
      - OrderNumber
      - Currency
      - ShippingAddressID
      - BillingAddressID
      - ShippingAddress
      - BillingAddress
      - Customer
      - ShippedAt
      - CancelledAt
      - CreatedAt
      - UpdatedAt

  # Nested mapping: store.OrderItem → warehouse.OrderItem
  - source: caster-generator/store.OrderItem
    target: caster-generator/warehouse.OrderItem

    # Requires OrderID from parent
    requires:
      - name: OrderID
        type: uint

    fields:
      # Use passed OrderID with identity transform
      - source: OrderID
        target: OrderID
        transform: store_transforms.PassOrderID

      # ProductID conversion
      - source: ProductID
        target: ProductID
        transform: store_transforms.Int64ToUint

    "121":
      Quantity: Quantity
      UnitPrice: UnitPrice

    ignore:
      - ID
      - TotalPrice
      - Order
      - Product
      - CreatedAt
      - UpdatedAt
EOF

show_yaml "${stage_dir}/phase3_patched.yaml" "Patched mapping with all features:"

prompt_continue

subheader "Phase 4: Create Transform Functions"

cat > "${stage_dir}/transforms.go" << 'EOF'
package store_transforms

import "caster-generator/store"

// Int64ToUint converts int64 to uint.
func Int64ToUint(v int64) uint {
    if v < 0 {
        return 0
    }
    return uint(v)
}

// OrderStatusToString converts OrderStatus enum to string.
func OrderStatusToString(s store.OrderStatus) string {
    return string(s)
}

// PassOrderID is a pass-through for context passing.
func PassOrderID(id uint) uint {
    return id
}
EOF

show_go "${stage_dir}/transforms.go" "Transform implementations:"

info "In a real project, this would be in your codebase."

prompt_continue

subheader "Phase 5: Generate Casters"

info "Running gen command..."
echo ""

run_cg gen \
    -mapping "${stage_dir}/phase3_patched.yaml" \
    -out "$out_dir" \
    -pkg ./store \
    -pkg ./warehouse \
    -package casters || warn "Some warnings may occur"

info "Generated files:"
ls -la "$out_dir" 2>/dev/null || echo "(check for errors)"

prompt_continue

subheader "Phase 6: Examine Generated Code"

for f in "$out_dir"/*.go; do
    if [[ -f "$f" ]]; then
        show_go "$f" "Generated: $(basename "$f")"
    fi
done

prompt_continue

subheader "Phase 7: Validate with Check"

info "Running check to validate..."
echo ""

if run_cg check \
    -mapping "${stage_dir}/phase3_patched.yaml" \
    -pkg ./store \
    -pkg ./warehouse 2>&1; then
    success "Mapping is valid!"
else
    warn "Check reported issues (may be expected for demo)"
fi

prompt_continue

subheader "Summary: Complete Workflow"

info "The full caster-generator workflow:"
echo ""
info "  1. ANALYZE  — Explore types: caster-generator analyze -pkg ./..."
info "  2. SUGGEST  — Get suggestions: caster-generator suggest -from A -to B"
info "  3. PATCH    — Edit YAML: add transforms, dives, ignores, requires"
info "  4. IMPROVE  — Re-run suggest: caster-generator suggest -mapping file.yaml"
info "  5. GENERATE — Create code: caster-generator gen mapping.yaml ./out"
info "  6. IMPLEMENT — Write transform functions"
info "  7. CHECK    — Validate: caster-generator check -mapping file.yaml"
info "  8. CI/CD    — Add check to your pipeline"
echo ""

done_step "18"

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}🎉 CONGRATULATIONS! You've completed the full tutorial!${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
echo ""
info "You now know how to:"
info "  ✅ Analyze packages and discover types"
info "  ✅ Generate mapping suggestions"
info "  ✅ Use 121, fields, ignore, and auto sections"
info "  ✅ Handle type mismatches with transforms"
info "  ✅ Map nested structs and collections"
info "  ✅ Handle pointers and recursive types"
info "  ✅ Pass context between casters"
info "  ✅ Generate virtual types"
info "  ✅ Validate mappings in CI/CD"
echo ""
info "For more details, see:"
info "  • GUIDE.md — This tutorial in written form"
info "  • FEATURES.md — Complete feature reference"
info "  • README.md — Quick start guide"
echo ""
info "Happy mapping! 🚀"

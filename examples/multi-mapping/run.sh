#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Multi-Mapping Example: Chain of type conversions
# ============================================================================
# Types: multi.ExternalOrder -> multi.InternalOrder -> multi.WarehouseOrder
#
# Goal: Demonstrate mapping chain across system boundaries
#       External API -> Internal Processing -> Warehouse Fulfillment
# ============================================================================

scenario_init "Multi-Mapping" "Demonstrate chained mappings across system boundaries"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

info "This example demonstrates a common pattern:"
echo "  External API Types -> Internal Domain Types -> Warehouse Types"
echo ""
echo "Type chain:"
echo "  ExternalOrder  →  InternalOrder  →  WarehouseOrder"
echo "  ExternalItem   →  InternalItem   →  WarehouseItem"
echo "  ExternalAddress → InternalAddress → WarehouseAddress"
echo ""

prompt_continue

# ============================================================================
# Stage 1: Suggest External -> Internal mappings
# ============================================================================
stage_start "Suggest External → Internal" "Generate mappings from external API to internal domain"

run_suggest "${stages_dir}/stage1_ext_to_int.yaml" \
  -pkg ./examples/multi-mapping \
  -from "caster-generator/examples/multi-mapping.ExternalOrder" \
  -to "caster-generator/examples/multi-mapping.InternalOrder" \
  -min-confidence 0.4

show_yaml_with_comments "${stages_dir}/stage1_ext_to_int.yaml" \
  "External -> Internal mapping suggestions"

info "Key observations (External -> Internal):"
echo "  • ExtOrderID -> ExternalRef (rename)"
echo "  • BuyerEmail -> CustomerEmail (rename)"
echo "  • BuyerName -> CustomerName (rename)"
echo "  • Items -> LineItems (dive, different element types)"
echo "  • ShipTo -> ShippingAddress (dive)"
echo "  • OrderDate (string) -> PlacedAt (time.Time) - needs parse"
echo "  • TotalAmount (float64) -> TotalCents (int64) - needs transform"
echo ""

prompt_continue

# ============================================================================
# Stage 2: Suggest Internal -> Warehouse mappings
# ============================================================================
stage_start "Suggest Internal → Warehouse" "Generate mappings from internal to warehouse"

run_suggest "${stages_dir}/stage2_int_to_wh.yaml" \
  -pkg ./examples/multi-mapping \
  -from "caster-generator/examples/multi-mapping.InternalOrder" \
  -to "caster-generator/examples/multi-mapping.WarehouseOrder" \
  -min-confidence 0.4

show_yaml_with_comments "${stages_dir}/stage2_int_to_wh.yaml" \
  "Internal -> Warehouse mapping suggestions"

info "Key observations (Internal -> Warehouse):"
echo "  • ID/ExternalRef -> OrderNumber (needs format)"
echo "  • CustomerName -> Recipient (rename)"
echo "  • LineItems -> Items (dive)"
echo "  • ShippingAddress -> Address (dive)"
echo "  • Priority is new field (may need default or ignore)"
echo ""

prompt_continue

# ============================================================================
# Stage 3: Create merged mapping file with all conversions
# ============================================================================
stage_start "Create Merged Mapping" "Combine all mappings with transforms"

cat > "${stages_dir}/stage3_merged.yaml" << 'EOF'
version: "1"

transforms:
  # Date/time parsing
  - name: ParseISODate
    signature: "func(s string) time.Time"
  # Currency conversion
  - name: DollarsToCents
    signature: "func(dollars float64) int64"
  # ID formatting
  - name: FormatOrderNumber
    signature: "func(id uint, ref string) string"
  # Address formatting
  - name: FormatFullAddress
    signature: "func(line1, city, state, postal, country string) string"

mappings:
  # ============================================
  # External -> Internal Order
  # ============================================
  - source: caster-generator/examples/multi-mapping.ExternalOrder
    target: caster-generator/examples/multi-mapping.InternalOrder
    fields:
      - source: ExtOrderID
        target: ExternalRef
      - source: BuyerEmail
        target: CustomerEmail
      - source: BuyerName
        target: CustomerName
      - source: Items
        target: LineItems
        hint: dive
      - source: ShipTo
        target: ShippingAddress
        hint: dive
      - source: OrderDate
        target: PlacedAt
        transform: ParseISODate
      - source: TotalAmount
        target: TotalCents
        transform: DollarsToCents
    ignore:
      - ID  # Internal ID is auto-generated

  - source: caster-generator/examples/multi-mapping.ExternalItem
    target: caster-generator/examples/multi-mapping.InternalItem
    fields:
      - source: ItemCode
        target: SKU
      - source: ItemName
        target: Name
      - source: Qty
        target: Quantity
      - source: UnitPrice
        target: PriceCents
        transform: DollarsToCents
    ignore:
      - ID  # Auto-generated

  - source: caster-generator/examples/multi-mapping.ExternalAddress
    target: caster-generator/examples/multi-mapping.InternalAddress
    fields:
      - source: Street
        target: Line1
      - source: City
        target: City
      - source: State
        target: State
      - source: Zip
        target: PostalCode
      - source: Country
        target: Country
    ignore:
      - ID

  # ============================================
  # Internal -> Warehouse Order
  # ============================================
  - source: caster-generator/examples/multi-mapping.InternalOrder
    target: caster-generator/examples/multi-mapping.WarehouseOrder
    fields:
      - source: [ID, ExternalRef]
        target: OrderNumber
        transform: FormatOrderNumber
      - source: CustomerName
        target: Recipient
      - source: LineItems
        target: Items
        hint: dive
      - source: ShippingAddress
        target: Address
        hint: dive
    ignore:
      - Priority  # Set by business logic, not mapping

  - source: caster-generator/examples/multi-mapping.InternalItem
    target: caster-generator/examples/multi-mapping.WarehouseItem
    fields:
      - source: SKU
        target: ProductCode
      - source: Name
        target: Description
      - source: Quantity
        target: Units

  - source: caster-generator/examples/multi-mapping.InternalAddress
    target: caster-generator/examples/multi-mapping.WarehouseAddress
    fields:
      - source: [Line1, City, State, PostalCode, Country]
        target: FullAddress
        transform: FormatFullAddress
      - source: City
        target: City
      - source: State
        target: Region
      - source: PostalCode
        target: ZIP
      - source: Country
        target: CountryCode
EOF

show_yaml_with_comments "${stages_dir}/stage3_merged.yaml" \
  "Complete mapping chain with transforms"

prompt_continue

# ============================================================================
# Stage 4: Generate transform implementations
# ============================================================================
stage_start "Generate Transforms" "Create transform function implementations"

cat > "${here}/transforms.go" << 'GOEOF'
package multi

import (
	"fmt"
	"strings"
	"time"
)

// ParseISODate parses an ISO date string (2006-01-02) to time.Time.
func ParseISODate(s string) time.Time {
	// Try common formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		time.RFC3339,
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// DollarsToCents converts dollars (float64) to cents (int64).
func DollarsToCents(dollars float64) int64 {
	return int64(dollars * 100)
}

// CentsToDollars converts cents (int64) to dollars (float64).
func CentsToDollars(cents int64) float64 {
	return float64(cents) / 100
}

// UintToString converts uint to string.
func UintToString(id uint) string {
	return fmt.Sprintf("%d", id)
}

// FormatOrderNumber creates a warehouse order number from internal ID and external ref.
func FormatOrderNumber(id uint, ref string) string {
	if ref != "" {
		return fmt.Sprintf("WH-%s", ref)
	}
	return fmt.Sprintf("WH-%d", id)
}

// FormatFullAddress combines address parts into a single line.
func FormatFullAddress(line1, city, state, postal, country string) string {
	parts := []string{}
	if line1 != "" {
		parts = append(parts, line1)
	}
	if city != "" {
		parts = append(parts, city)
	}
	if state != "" {
		parts = append(parts, state)
	}
	if postal != "" {
		parts = append(parts, postal)
	}
	if country != "" {
		parts = append(parts, country)
	}
	return strings.Join(parts, ", ")
}
GOEOF

show_file "${here}/transforms.go" "Transform implementations"

stage_done "Transform functions created"

prompt_continue

# ============================================================================
# Stage 5: Validate mapping
# ============================================================================
stage_start "Validate Mapping" "Run suggest to validate structure"

run_suggest_improve "${stages_dir}/stage3_merged.yaml" "${stages_dir}/stage5_validated.yaml" \
  -pkg ./examples/multi-mapping \
  -min-confidence 0.4

show_yaml_with_comments "${stages_dir}/stage5_validated.yaml" \
  "Validated mapping"

prompt_continue

# ============================================================================
# Stage 6: Generate casters
# ============================================================================
stage_start "Generate Casters" "Generate all caster functions"

# Use the merged file which has the transforms section
run_gen "${stages_dir}/stage3_merged.yaml" "$out_dir" \
  -pkg ./examples/multi-mapping \
  -package casters || true

info "Generated files:"
ls -la "$out_dir"

# Check if we got any files
file_count=$(find "$out_dir" -name "*.go" 2>/dev/null | wc -l)
if [[ "$file_count" -eq 0 ]]; then
  warn "No casters generated - this example requires dive hint support for nested structs"
  warn "The mapping file demonstrates the structure for chained mappings"
  scenario_partial "Generation requires improved dive hint handling for nested struct mapping"
  exit 1
fi

prompt_continue

# ============================================================================
# Stage 7: Compile check
# ============================================================================
stage_start "Compile Check" "Verify everything compiles"

test_compile ./examples/multi-mapping || {
  warn "Compile check failed"
  scenario_partial "Review errors and adjust transforms"
  exit 1
}

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping: ${stages_dir}/stage5_validated.yaml"
info "Transform functions: ${here}/transforms.go"
info "Generated casters: ${out_dir}/"
echo ""
info "Mapping chain demonstrated:"
echo "  ExternalOrder  →  InternalOrder  →  WarehouseOrder"
echo "  ExternalItem   →  InternalItem   →  WarehouseItem"
echo "  ExternalAddress → InternalAddress → WarehouseAddress"

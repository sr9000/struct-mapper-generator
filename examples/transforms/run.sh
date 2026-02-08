#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Transforms Example: Type incompatibilities with custom transforms
# ============================================================================
# Types: transforms.LegacyProduct -> transforms.ModernProduct
#        transforms.LegacyUser -> transforms.ModernUser
#
# Goal: Canonical multi-iteration scenario for type incompatibilities.
#       Demonstrates placeholder transforms generated on the fly.
# ============================================================================

scenario_init "Transforms" "Demonstrate custom transforms for type conversions"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

info "This example shows the complete workflow for handling type incompatibilities:"
echo "  1. Initial suggest identifies field candidates"
echo "  2. Manual mapping patches lock in intent"
echo "  3. Generator identifies needed transforms (placeholders)"
echo "  4. Transform stubs are generated on the fly"
echo "  5. User implements transform logic"
echo "  6. Final generation produces working casters"
echo ""

prompt_continue

# ============================================================================
# Stage 1: Suggest with lower confidence (catch more candidates)
# ============================================================================
stage_start "Initial Suggest (Low Confidence)" "Generate mappings with lower threshold to see more candidates"

run_suggest "${stages_dir}/stage1_product.yaml" \
  -pkg ./examples/transforms \
  -from "caster-generator/examples/transforms.LegacyProduct" \
  -to "caster-generator/examples/transforms.ModernProduct" \
  -min-confidence 0.4

show_yaml_with_comments "${stages_dir}/stage1_product.yaml" \
  "Product mapping - notice many fields need transforms"

info "Type incompatibilities detected:"
echo "  • Code (string) -> SKU (string) - simple rename"
echo "  • Title (string) -> Name (string) - simple rename"
echo "  • PriceUSD (float64) -> PriceCents (int64) - needs transform"
echo "  • Weight (string '2.5kg') -> WeightGrams (int) - needs parse transform"
echo "  • Available (string 'Y'/'N') -> IsActive (bool) - needs transform"
echo "  • LastModified (string) -> UpdatedAt (time.Time) - needs parse transform"
echo "  • Categories (string CSV) -> Tags ([]string) - needs split transform"
echo ""

prompt_continue

run_suggest "${stages_dir}/stage1_user.yaml" \
  -pkg ./examples/transforms \
  -from "caster-generator/examples/transforms.LegacyUser" \
  -to "caster-generator/examples/transforms.ModernUser" \
  -min-confidence 0.4

show_yaml_with_comments "${stages_dir}/stage1_user.yaml" \
  "User mapping - multiple transforms needed"

info "User type incompatibilities:"
echo "  • UserID (int) -> ID (uint) - needs cast transform"
echo "  • FirstName + LastName -> FullName - needs concat transform"
echo "  • PhoneNumber (string) -> Phone (string) - needs normalize transform"
echo "  • BirthDate (string) -> DateOfBirth (*time.Time) - needs parse transform"
echo "  • Address (JSON string) -> City, PostalCode - needs JSON parse transform"
echo ""

prompt_continue

# ============================================================================
# Stage 2: Merge and create explicit mapping with transforms
# ============================================================================
stage_start "Create Explicit Mapping" "Lock in field mappings with transform declarations"

cat > "${stages_dir}/stage2_merged.yaml" << 'EOF'
version: "1"

# Transform function declarations
# Signatures describe expected input/output types
transforms:
  # Product transforms
  - name: DollarsToCents
    signature: "func(dollars float64) int64"
  - name: ParseWeightGrams
    signature: "func(weight string) int"
  - name: YNToBool
    signature: "func(yn string) bool"
  - name: ParseDateTime
    signature: "func(s string) time.Time"
  - name: CSVToSlice
    signature: "func(csv string) []string"

  # User transforms
  - name: IntToUint
    signature: "func(i int) uint"
  - name: ConcatNames
    signature: "func(first, last string) string"
  - name: NormalizePhone
    signature: "func(phone string) string"
  - name: ParseDate
    signature: "func(s string) *time.Time"
  - name: ExtractCity
    signature: "func(jsonAddr string) string"
  - name: ExtractPostalCode
    signature: "func(jsonAddr string) string"

mappings:
  # LegacyProduct -> ModernProduct
  - source: caster-generator/examples/transforms.LegacyProduct
    target: caster-generator/examples/transforms.ModernProduct
    fields:
      - source: Code
        target: SKU
      - source: Title
        target: Name
      - source: PriceUSD
        target: PriceCents
        transform: DollarsToCents
      - source: Weight
        target: WeightGrams
        transform: ParseWeightGrams
      - source: Available
        target: IsActive
        transform: YNToBool
      - source: LastModified
        target: UpdatedAt
        transform: ParseDateTime
      - source: Categories
        target: Tags
        transform: CSVToSlice

  # LegacyUser -> ModernUser
  - source: caster-generator/examples/transforms.LegacyUser
    target: caster-generator/examples/transforms.ModernUser
    fields:
      - source: UserID
        target: ID
        transform: IntToUint
      - source: [FirstName, LastName]
        target: FullName
        transform: ConcatNames
      - source: PhoneNumber
        target: Phone
        transform: NormalizePhone
      - source: BirthDate
        target: DateOfBirth
        transform: ParseDate
      - source: Address
        target: City
        transform: ExtractCity
      - source: Address
        target: PostalCode
        transform: ExtractPostalCode
EOF

show_yaml_with_comments "${stages_dir}/stage2_merged.yaml" \
  "Complete mapping with all transform declarations"

info "Transform functions declared:"
echo "  Product: DollarsToCents, ParseWeightGrams, YNToBool, ParseDateTime, CSVToSlice"
echo "  User: IntToUint, ConcatNames, NormalizePhone, ParseDate, ExtractCity, ExtractPostalCode"
echo ""

prompt_continue

# ============================================================================
# Stage 3: Suggest improve to validate
# ============================================================================
stage_start "Suggest Improve" "Validate mapping structure"

run_suggest_improve "${stages_dir}/stage2_merged.yaml" "${stages_dir}/stage3_validated.yaml" \
  -pkg ./examples/transforms \
  -min-confidence 0.4

show_yaml_with_comments "${stages_dir}/stage3_validated.yaml" \
  "Validated mapping structure (note: using merged file for gen since it has transforms)"

# The suggest command doesn't preserve transforms section, so we use the merged file
info "Using merged mapping for generation (contains transform declarations)"

prompt_continue

# ============================================================================
# Stage 4: Generate transform stubs on the fly
# ============================================================================
stage_start "Generate Transform Stubs" "Auto-generate Go file with transform function stubs"

cat > "${here}/transforms.go" << 'GOEOF'
package transforms

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// Product Transforms
// ============================================================================

// DollarsToCents converts a dollar amount (float64) to cents (int64).
func DollarsToCents(dollars float64) int64 {
	return int64(dollars * 100)
}

// ParseWeightGrams parses a weight string like "2.5kg" to grams.
func ParseWeightGrams(weight string) int {
	weight = strings.ToLower(strings.TrimSpace(weight))

	// Try to extract numeric part
	re := regexp.MustCompile(`^([\d.]+)\s*(kg|g|lb|oz)?$`)
	matches := re.FindStringSubmatch(weight)
	if len(matches) < 2 {
		return 0
	}

	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := "g"
	if len(matches) >= 3 {
		unit = matches[2]
	}

	switch unit {
	case "kg":
		return int(val * 1000)
	case "lb":
		return int(val * 453.592)
	case "oz":
		return int(val * 28.3495)
	default:
		return int(val)
	}
}

// YNToBool converts "Y"/"N" string to bool.
func YNToBool(yn string) bool {
	return strings.ToUpper(strings.TrimSpace(yn)) == "Y"
}

// ParseDateTime parses a datetime string in "2006-01-02 15:04:05" format.
func ParseDateTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// CSVToSlice splits a comma-separated string into a slice.
func CSVToSlice(csv string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ============================================================================
// User Transforms
// ============================================================================

// IntToUint converts int to uint.
func IntToUint(i int) uint {
	if i < 0 {
		return 0
	}
	return uint(i)
}

// ConcatNames combines first and last name into full name.
func ConcatNames(first, last string) string {
	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)
	if first == "" {
		return last
	}
	if last == "" {
		return first
	}
	return first + " " + last
}

// NormalizePhone normalizes a phone number to digits only.
func NormalizePhone(phone string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(phone, "")
}

// ParseDate parses a date string in "2006-01-02" format.
func ParseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

// addressJSON is a helper struct for parsing address JSON.
type addressJSON struct {
	City string `json:"city"`
	Zip  string `json:"zip"`
}

// ExtractCity extracts city from a JSON address string.
func ExtractCity(jsonAddr string) string {
	var addr addressJSON
	if err := json.Unmarshal([]byte(jsonAddr), &addr); err != nil {
		return ""
	}
	return addr.City
}

// ExtractPostalCode extracts postal code from a JSON address string.
func ExtractPostalCode(jsonAddr string) string {
	var addr addressJSON
	if err := json.Unmarshal([]byte(jsonAddr), &addr); err != nil {
		return ""
	}
	return addr.Zip
}
GOEOF

show_file "${here}/transforms.go" "Generated transform implementations"

stage_done "Transform functions generated on the fly"

info "All transform functions are now implemented:"
echo "  • DollarsToCents, ParseWeightGrams, YNToBool, ParseDateTime, CSVToSlice"
echo "  • IntToUint, ConcatNames, NormalizePhone, ParseDate, ExtractCity, ExtractPostalCode"
echo ""

prompt_continue

# ============================================================================
# Stage 5: Generate casters
# ============================================================================
stage_start "Generate Casters" "Generate final caster code"

# Use the merged file which has the transforms section
run_gen "${stages_dir}/stage2_merged.yaml" "$out_dir" \
  -pkg ./examples/transforms \
  -package casters

info "Generated files:"
ls -la "$out_dir"

prompt_continue

# ============================================================================
# Stage 6: Compile check
# ============================================================================
stage_start "Compile Check" "Verify everything compiles together"

test_compile ./examples/transforms || {
  warn "Compile check failed - reviewing errors..."
  scenario_partial "Check transform function signatures match generated code"
  exit 0
}

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping: ${stages_dir}/stage2_merged.yaml"
info "Transform functions: ${here}/transforms.go"
info "Generated casters: ${out_dir}/"
echo ""
info "Summary of transforms demonstrated:"
echo "  • Numeric conversions: float64->int64 (cents), int->uint"
echo "  • String parsing: weight, date, datetime"
echo "  • Boolean conversion: Y/N -> bool"
echo "  • String splitting: CSV -> []string"
echo "  • Multi-source: FirstName + LastName -> FullName"
echo "  • JSON extraction: Address JSON -> City, PostalCode"

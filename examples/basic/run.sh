#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=../_scripts/common.sh
source "${here}/../_scripts/common.sh"

# ============================================================================
# Basic Example: Renames and numeric type changes
# ============================================================================
# Types: basic.UserDTO -> basic.User
#        basic.OrderDTO -> basic.Order
#
# Goal: Show iterative improvement where suggest gets easy renames,
#       then placeholders for type conversions (int64->uint, float64->int64)
# ============================================================================

scenario_init "Basic" "Demonstrate field renames and numeric type conversions with transforms"

# Directories
stages_dir="${here}/stages"
out_dir="${here}/generated"
clean_dir "$stages_dir"
clean_dir "$out_dir"

# ============================================================================
# Stage 1: Initial suggest for UserDTO -> User
# ============================================================================
stage_start "Suggest UserDTO → User" "Generate initial mapping for user type"

run_suggest "${stages_dir}/stage1_user.yaml" \
  -pkg ./examples/basic \
  -from "caster-generator/examples/basic.UserDTO" \
  -to "caster-generator/examples/basic.User"

show_yaml_with_comments "${stages_dir}/stage1_user.yaml" \
  "Notice: ID (int64) -> UserID (uint) needs transform, FullName -> Name rename"

prompt_continue

# ============================================================================
# Stage 2: Initial suggest for OrderDTO -> Order
# ============================================================================
stage_start "Suggest OrderDTO → Order" "Generate initial mapping for order type"

run_suggest "${stages_dir}/stage2_order.yaml" \
  -pkg ./examples/basic \
  -from "caster-generator/examples/basic.OrderDTO" \
  -to "caster-generator/examples/basic.Order"

show_yaml_with_comments "${stages_dir}/stage2_order.yaml" \
  "Notice: Amount (float64) -> TotalCents (int64) needs transform"

prompt_continue

# ============================================================================
# Stage 3: Merge and patch mappings
# ============================================================================
stage_start "Merge & Patch" "Combine mappings and add explicit field mappings with transforms"

cat > "${stages_dir}/stage3_merged.yaml" << 'EOF'
version: "1"

# Transform function declarations
# These need to be implemented in your code
transforms:
  - name: Int64ToUint
    signature: "func(v int64) uint"
  - name: Float64ToCents
    signature: "func(dollars float64) int64"
  - name: StringToUint
    signature: "func(s string) uint"

mappings:
  # UserDTO -> User mapping
  - source: caster-generator/examples/basic.UserDTO
    target: caster-generator/examples/basic.User
    fields:
      # ID needs type conversion: int64 -> uint
      - source: ID
        target: UserID
        transform: Int64ToUint
      # FullName -> Name is a simple rename
      - source: FullName
        target: Name
      # IsActive -> Active is a simple rename
      - source: IsActive
        target: Active
    auto:
      # These match directly
      - source: Email
        target: Email
      - source: CreatedAt
        target: CreatedAt

  # OrderDTO -> Order mapping
  - source: caster-generator/examples/basic.OrderDTO
    target: caster-generator/examples/basic.Order
    fields:
      # OrderID (string) -> ID (uint) needs transform
      - source: OrderID
        target: ID
        transform: StringToUint
      # CustomerID: int64 -> uint needs transform
      - source: CustomerID
        target: CustomerID
        transform: Int64ToUint
      # Amount (float64 dollars) -> TotalCents (int64 cents)
      - source: Amount
        target: TotalCents
        transform: Float64ToCents
    auto:
      - source: Status
        target: Status
EOF

show_yaml_with_comments "${stages_dir}/stage3_merged.yaml" \
  "Combined mapping with explicit transforms for type conversions"

info "Transform functions needed:"
echo "  • Int64ToUint: func(v int64) uint"
echo "  • Float64ToCents: func(dollars float64) int64"
echo "  • StringToUint: func(s string) uint"
echo ""

prompt_continue

# ============================================================================
# Stage 4: Suggest improve (validate our manual mapping)
# ============================================================================
stage_start "Suggest Improve" "Run suggest with our mapping to validate and improve"

run_suggest_improve "${stages_dir}/stage3_merged.yaml" "${stages_dir}/stage4_improved.yaml" \
  -pkg ./examples/basic

show_yaml_with_comments "${stages_dir}/stage4_improved.yaml" \
  "Validated mapping (note: transforms section preserved from merged file)"

# Copy transforms section from merged to improved (suggest doesn't preserve it)
# We use the merged file for generation since it has the transform declarations
info "Using merged mapping for generation (contains transform declarations)"

prompt_continue

# ============================================================================
# Stage 5: Generate transform stubs
# ============================================================================
stage_start "Generate Transform Stubs" "Create Go file with transform function stubs"

cat > "${here}/transforms.go" << 'EOF'
package basic

import (
	"strconv"
)

// Transform functions for basic type conversions.
// These are called by the generated caster functions.

// Int64ToUint converts int64 to uint.
// Note: This will panic on negative values in production code.
func Int64ToUint(v int64) uint {
	if v < 0 {
		panic("cannot convert negative int64 to uint")
	}
	return uint(v)
}

// Float64ToCents converts a dollar amount (float64) to cents (int64).
func Float64ToCents(dollars float64) int64 {
	return int64(dollars * 100)
}

// StringToUint parses a string as uint.
func StringToUint(s string) uint {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic("cannot parse string as uint: " + err.Error())
	}
	return uint(v)
}
EOF

show_file "${here}/transforms.go" "Transform implementations"

stage_done "Transform functions created in transforms.go"

prompt_continue

# ============================================================================
# Stage 6: Generate code
# ============================================================================
stage_start "Generate Code" "Generate caster functions"

# Use the merged file which has the transforms section
run_gen "${stages_dir}/stage3_merged.yaml" "$out_dir" \
  -pkg ./examples/basic \
  -package casters

info "Generated files:"
ls -la "$out_dir"

prompt_continue

# ============================================================================
# Stage 7: Compile check
# ============================================================================
stage_start "Compile Check" "Verify generated code compiles with transforms"

test_compile ./examples/basic

# ============================================================================
# Done
# ============================================================================
scenario_success

info "Final mapping: ${stages_dir}/stage3_merged.yaml"
info "Transform functions: ${here}/transforms.go"
info "Generated casters: ${out_dir}/"

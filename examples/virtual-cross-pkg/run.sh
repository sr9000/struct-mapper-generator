#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/../_scripts/common.sh"

scenario_init "Virtual Cross-Pkg" "Generate target struct in a different package"

clean_dir "${here}/stages"
clean_dir "${here}/generated"
# Clean missing_types.go if it exists from previous run
rm -f "${here}/target/missing_types.go"

# Mapping
mkdir -p "${here}/stages"
cat > "${here}/stages/mapping.yaml" << 'EOF'
version: "1"
mappings:
  - source: caster-generator/examples/virtual-cross-pkg/source.Input
    target: caster-generator/examples/virtual-cross-pkg/target.Output
    generate_target: true
    fields:
      - source: Data
        target: Data
EOF

# Explicitly load target package so generator knows where it is
run_gen "${here}/stages/mapping.yaml" "${here}/generated" \
  -pkg caster-generator/examples/virtual-cross-pkg/source \
  -pkg caster-generator/examples/virtual-cross-pkg/target

# Verification
if [[ -f "${here}/target/missing_types.go" ]]; then
  info "✓ missing_types.go generated in target package"
else
  fail "missing_types.go not found in ${here}/target/"
fi

if grep -q "package target" "${here}/target/missing_types.go"; then
  info "✓ Correct package name"
else
  fail "Incorrect package name"
fi

if grep -q "type Output struct" "${here}/target/missing_types.go"; then
  info "✓ Output struct definition found"
else
  fail "Output struct definition not found"
fi

stage_start "Compile Check" "Verify generated code compiles"
test_compile ./examples/virtual-cross-pkg/target
# We need to test compile generated code too, which imports target
# Note: generated code is in a separate package 'casters', so we need something to consume it or just check it builds
# test_compile checks 'go test -c'

# Fix: generated code might not be compile-able in isolation if it depends on 'target' package within the same module
# but we are in workspace mode usually?
# Let's try compiling the directory
if go build -o /dev/null "${here}/generated"/*.go; then
    info "✓ Generated casters compile"
else
    fail "Generated casters failed to compile"
fi

scenario_success

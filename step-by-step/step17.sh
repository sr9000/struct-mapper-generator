#!/usr/bin/env bash
# Step 17: Check Command — CI/CD Validation
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 17: Check Command — CI/CD Validation"

info "The 'check' command validates mappings against actual code."
info "Use it in CI/CD to catch drift between code and mappings."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step17")

subheader "1. Creating a valid mapping"

cat > "${stage_dir}/valid_mapping.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIAddress
    target: caster-generator/step-by-step/tutorial.DomainAddress
    "121":
      Street: Street
      City: City
      ZipCode: PostalCode
      Country: Country
EOF

show_yaml "${stage_dir}/valid_mapping.yaml" "Valid mapping to check:"

prompt_continue

subheader "2. Running check command"

info "Running: caster-generator check -mapping valid_mapping.yaml"
echo ""

if run_cg check \
    -mapping "${stage_dir}/valid_mapping.yaml" \
    -pkg ./step-by-step/tutorial; then
    success "Check passed! Mapping is valid."
else
    error "Check failed!"
fi

prompt_continue

subheader "3. Creating an invalid mapping"

cat > "${stage_dir}/invalid_mapping.yaml" << 'EOF'
version: "1"

mappings:
  - source: caster-generator/step-by-step/tutorial.APIAddress
    target: caster-generator/step-by-step/tutorial.DomainAddress
    "121":
      Street: Street
      City: City
      ZipCode: PostalCode
      Country: Country
      # NonExistentField is not in the source type!
      NonExistentField: SomeTarget
EOF

show_yaml "${stage_dir}/invalid_mapping.yaml" "Invalid mapping (field doesn't exist):"

prompt_continue

subheader "4. Check detects the error"

info "Running check on invalid mapping..."
echo ""

if run_cg check \
    -mapping "${stage_dir}/invalid_mapping.yaml" \
    -pkg ./step-by-step/tutorial 2>&1; then
    warn "Check unexpectedly passed"
else
    info "Check correctly detected the invalid field!"
fi

prompt_continue

subheader "5. CI/CD integration"

info "Example GitHub Actions workflow:"
echo ""
cat << 'EOF'
# .github/workflows/check-mappings.yml
name: Check Mappings

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build caster-generator
        run: make build

      - name: Validate mappings
        run: |
          ./caster-generator check \
            -mapping mappings/orders.yaml \
            -pkg ./...

      - name: Strict mode (all fields must be mapped)
        run: |
          ./caster-generator check \
            -mapping mappings/orders.yaml \
            -pkg ./... \
            -strict
EOF

echo ""
info "The check command catches:"
info "  • Renamed or removed fields"
info "  • Type changes"
info "  • Missing types"
info "  • Invalid transform references"

done_step "17"
info "Files are in: ${stage_dir}/"
info "Next: ./step18.sh — Complete real-world workflow"

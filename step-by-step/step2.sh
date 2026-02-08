#!/usr/bin/env bash
# Step 2: Suggest — Getting Smart Mapping Suggestions
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 2: Suggest — Getting Smart Mapping Suggestions"

info "The 'suggest' command uses intelligent matching to propose field mappings."
info "It analyzes field names, types, and structure to find likely matches."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step2")

subheader "1. Generate initial suggestion"

info "Running: caster-generator suggest -pkg ./step-by-step/tutorial \\"
info "    -from tutorial.APIProduct -to tutorial.DomainProduct"
echo ""

run_cg suggest \
    -pkg ./step-by-step/tutorial \
    -from "caster-generator/step-by-step/tutorial.APIProduct" \
    -to "caster-generator/step-by-step/tutorial.DomainProduct" \
    -out "${stage_dir}/product_mapping.yaml"

show_yaml "${stage_dir}/product_mapping.yaml" "Generated mapping suggestion:"

info "Notice the 'auto' section with confidence scores."
info "Fields with low confidence may need manual adjustment."

prompt_continue

subheader "2. Lower confidence threshold"

info "Lower threshold (0.4) catches more matches but may include false positives."
info "Running with -min-confidence 0.4"
echo ""

run_cg suggest \
    -pkg ./step-by-step/tutorial \
    -from "caster-generator/step-by-step/tutorial.APICustomer" \
    -to "caster-generator/step-by-step/tutorial.DomainCustomer" \
    -min-confidence 0.4 \
    -out "${stage_dir}/customer_mapping_low.yaml"

show_yaml "${stage_dir}/customer_mapping_low.yaml" "Low confidence mapping:"

prompt_continue

subheader "3. Higher confidence threshold"

info "Higher threshold (0.9) is stricter — fewer but more reliable matches."
info "Running with -min-confidence 0.9"
echo ""

run_cg suggest \
    -pkg ./step-by-step/tutorial \
    -from "caster-generator/step-by-step/tutorial.APICustomer" \
    -to "caster-generator/step-by-step/tutorial.DomainCustomer" \
    -min-confidence 0.9 \
    -out "${stage_dir}/customer_mapping_high.yaml"

show_yaml "${stage_dir}/customer_mapping_high.yaml" "High confidence mapping:"

info "Compare the two files — notice how threshold affects matches."

done_step "2"
info "Generated mappings are in: ${stage_dir}/"
info "Next: ./step3.sh — Generate caster code"

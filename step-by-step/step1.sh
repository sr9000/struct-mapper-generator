#!/usr/bin/env bash
# Step 1: Analyze — Exploring Your Types
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${here}/_common.sh"

header "Step 1: Analyze — Exploring Your Types"

info "The 'analyze' command lets you inspect packages and discover struct types."
info "This is your starting point for any mapping project."
echo ""

# Setup stage directory
stage_dir=$(setup_stage "step1")

subheader "1. Analyze the tutorial package"

info "Running: caster-generator analyze -pkg ./step-by-step/tutorial"
echo ""

run_cg analyze -pkg ./step-by-step/tutorial

prompt_continue

subheader "2. Filter to specific type"

info "Running: caster-generator analyze -pkg ./step-by-step/tutorial -type APIOrder"
echo ""

run_cg analyze -pkg ./step-by-step/tutorial -type APIOrder

prompt_continue

subheader "3. Verbose mode (shows tags)"

info "Running: caster-generator analyze -pkg ./step-by-step/tutorial -type APIProduct -verbose"
echo ""

run_cg analyze -pkg ./step-by-step/tutorial -type APIProduct -verbose

prompt_continue

subheader "4. Analyze store and warehouse packages"

info "Running: caster-generator analyze -pkg ./store -pkg ./warehouse"
echo ""

run_cg analyze -pkg ./store -pkg ./warehouse

done_step "1"
info "You've learned how to explore packages and discover types!"
info "Next: ./step2.sh — Generate mapping suggestions"

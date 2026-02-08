#!/usr/bin/env bash
# Common helpers for step-by-step tutorial scripts.
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Get repo root
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TUTORIAL_DIR="${REPO_ROOT}/step-by-step"
STAGES_DIR="${TUTORIAL_DIR}/stages"

# Run caster-generator from repo root
run_cg() {
    (cd "$REPO_ROOT" && go run ./cmd/caster-generator "$@")
}

# Clean and create a stage directory
setup_stage() {
    local stage_name="$1"
    local stage_dir="${STAGES_DIR}/${stage_name}"
    rm -rf "$stage_dir"
    mkdir -p "$stage_dir"
    echo "$stage_dir"
}

# Display a header
header() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC} ${YELLOW}$1${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Display a subheader
subheader() {
    echo ""
    echo -e "${BLUE}┌──────────────────────────────────────────────────────────────────┐${NC}"
    echo -e "${BLUE}│${NC} ${CYAN}$1${NC}"
    echo -e "${BLUE}└──────────────────────────────────────────────────────────────────┘${NC}"
}

# Display info message
info() {
    echo -e "${CYAN}ℹ ${NC}$*"
}

# Display success message
success() {
    echo -e "${GREEN}✓ ${NC}$*"
}

# Display warning message
warn() {
    echo -e "${YELLOW}⚠ ${NC}$*"
}

# Display error message
error() {
    echo -e "${RED}✗ ${NC}$*"
}

# Show a file with syntax highlighting (or cat if no bat)
show_file() {
    local file="$1"
    local title="${2:-$file}"

    echo ""
    echo -e "${CYAN}── ${title} ──${NC}"
    if command -v bat &> /dev/null; then
        bat --style=plain --language="${file##*.}" "$file" 2>/dev/null || cat "$file"
    else
        cat "$file"
    fi
    echo -e "${CYAN}────────────────────────────${NC}"
}

# Show YAML content with comment
show_yaml() {
    local file="$1"
    local comment="${2:-}"

    echo ""
    if [[ -n "$comment" ]]; then
        info "$comment"
    fi
    echo -e "${CYAN}── YAML: $(basename "$file") ──${NC}"
    cat "$file"
    echo -e "${CYAN}────────────────────────────${NC}"
}

# Show generated Go code
show_go() {
    local file="$1"
    local comment="${2:-}"

    echo ""
    if [[ -n "$comment" ]]; then
        info "$comment"
    fi
    echo -e "${CYAN}── Go: $(basename "$file") ──${NC}"
    cat "$file"
    echo -e "${CYAN}────────────────────────────${NC}"
}

# Wait for user to press enter (unless CG_NO_PROMPT is set)
prompt_continue() {
    if [[ "${CG_NO_PROMPT:-}" != "1" ]]; then
        echo ""
        read -r -p "Press Enter to continue..."
    fi
}

# Compile check
test_compile() {
    local pkg="$1"
    if (cd "$REPO_ROOT" && go build "$pkg" 2>/dev/null); then
        success "Compile check passed for $pkg"
        return 0
    else
        error "Compile check failed for $pkg"
        return 1
    fi
}

# Final success message
done_step() {
    local step="$1"
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}✓ Step $step completed successfully!${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
}

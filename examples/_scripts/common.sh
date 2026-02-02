#!/usr/bin/env bash
set -euo pipefail

# Common helpers for example run.sh scripts.
# Supports multi-stage scenarios with suggest → patch → generate workflow.

# ============================================================================
# Configuration
# ============================================================================

# Set CG_NO_PROMPT=1 to skip interactive prompts (for CI)
# Set CG_DEBUG=1 to show executed commands
# Set CG_VERBOSE=1 to show more output

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ============================================================================
# Core Helpers
# ============================================================================

repo_root() {
  # Resolve repo root from this script location.
  # examples/_scripts/common.sh -> repo root is two levels up.
  local d
  d="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  echo "$d"
}

run_caster_generator() {
  local root
  root="$(repo_root)"

  if [[ "${CG_DEBUG:-}" != "" ]]; then
    echo -e "${CYAN}+ (cd \"$root\" && go run ./cmd/caster-generator $*)${NC}" >&2
  fi

  (cd "$root" && go run ./cmd/caster-generator "$@")
}

clean_dir() {
  local dir="$1"
  rm -rf "$dir"
  mkdir -p "$dir"
}

# ============================================================================
# Stage Management
# ============================================================================

# Global stage counter
_STAGE_NUM=0

# Initialize a scenario run
scenario_init() {
  local name="$1"
  local desc="${2:-}"
  _STAGE_NUM=0

  echo ""
  echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║${NC} ${YELLOW}Scenario: ${name}${NC}"
  if [[ -n "$desc" ]]; then
    echo -e "${GREEN}║${NC} ${desc}"
  fi
  echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
  echo ""
}

# Start a new stage
stage_start() {
  local name="$1"
  local desc="${2:-}"
  ((_STAGE_NUM++)) || true

  echo ""
  echo -e "${BLUE}┌──────────────────────────────────────────────────────────────────┐${NC}"
  echo -e "${BLUE}│${NC} ${YELLOW}Stage ${_STAGE_NUM}: ${name}${NC}"
  if [[ -n "$desc" ]]; then
    echo -e "${BLUE}│${NC} ${desc}"
  fi
  echo -e "${BLUE}└──────────────────────────────────────────────────────────────────┘${NC}"
}

# Show stage completion
stage_done() {
  local msg="${1:-Stage completed}"
  echo -e "${GREEN}✓ ${msg}${NC}"
}

# Show info message
info() {
  echo -e "${CYAN}ℹ ${NC}$*"
}

# Show warning
warn() {
  echo -e "${YELLOW}⚠ ${NC}$*"
}

# Show error
error() {
  echo -e "${RED}✗ ${NC}$*" >&2
}

# ============================================================================
# Interactive Prompts
# ============================================================================

# Wait for user to continue (skipped if CG_NO_PROMPT=1)
prompt_continue() {
  local msg="${1:-Press Enter to continue...}"
  if [[ "${CG_NO_PROMPT:-}" == "1" ]]; then
    return 0
  fi
  echo ""
  echo -e "${YELLOW}${msg}${NC}"
  read -r
}

# Ask yes/no question (returns 0 for yes, 1 for no)
# In non-interactive mode, defaults to yes
prompt_yn() {
  local msg="$1"
  local default="${2:-y}"

  if [[ "${CG_NO_PROMPT:-}" == "1" ]]; then
    [[ "$default" == "y" ]] && return 0 || return 1
  fi

  local prompt
  if [[ "$default" == "y" ]]; then
    prompt="[Y/n]"
  else
    prompt="[y/N]"
  fi

  echo -n -e "${YELLOW}${msg} ${prompt}: ${NC}"
  read -r answer
  answer="${answer:-$default}"
  [[ "$answer" =~ ^[Yy] ]]
}

# ============================================================================
# File Display & Diff
# ============================================================================

# Show file contents with syntax highlighting hint
show_file() {
  local file="$1"
  local title="${2:-$file}"

  echo ""
  echo -e "${CYAN}─── ${title} ───${NC}"
  if [[ -f "$file" ]]; then
    cat "$file"
  else
    echo "(file does not exist)"
  fi
  echo -e "${CYAN}─── end ───${NC}"
  echo ""
}

# Show diff between two files (if diff exists)
show_diff() {
  local file1="$1"
  local file2="$2"
  local title="${3:-Diff: $file1 → $file2}"

  echo ""
  echo -e "${CYAN}─── ${title} ───${NC}"
  if ! diff -u "$file1" "$file2" 2>/dev/null; then
    echo "(files are identical or one doesn't exist)"
  fi
  echo -e "${CYAN}─── end diff ───${NC}"
  echo ""
}

# Show YAML file with comments about what to look for
show_yaml_with_comments() {
  local file="$1"
  local comment="${2:-}"

  if [[ -n "$comment" ]]; then
    echo ""
    echo -e "${YELLOW}# ${comment}${NC}"
  fi
  show_file "$file" "YAML: $(basename "$file")"
}

# ============================================================================
# YAML Patching Helpers
# ============================================================================

# Apply a sed-style patch to a YAML file
# Usage: yaml_patch input.yaml output.yaml 's/old/new/'
yaml_patch() {
  local input="$1"
  local output="$2"
  shift 2
  sed "$@" "$input" > "$output"
}

# Add a line after a matching pattern
# Usage: yaml_add_after input.yaml output.yaml 'pattern' 'new line'
yaml_add_after() {
  local input="$1"
  local output="$2"
  local pattern="$3"
  local newline="$4"
  sed "/${pattern}/a\\${newline}" "$input" > "$output"
}

# Add hint: dive to a field mapping by target name
# Usage: yaml_add_dive_hint input.yaml output.yaml 'FieldName'
yaml_add_dive_hint() {
  local input="$1"
  local output="$2"
  local field="$3"
  # Add 'hint: dive' after 'target: FieldName' line
  sed "/target: ${field}$/a\\        hint: dive" "$input" > "$output"
}

# Append content to YAML file
yaml_append() {
  local file="$1"
  local content="$2"
  echo "$content" >> "$file"
}

# ============================================================================
# Suggest & Generate Workflow
# ============================================================================

# Run suggest command and save output
# Usage: run_suggest output.yaml [extra_args...]
run_suggest() {
  local output="$1"
  shift

  info "Running suggest → ${output}"
  run_caster_generator suggest -out "$output" "$@"
  stage_done "Suggestions saved to ${output}"
}

# Run suggest with existing mapping (improve mode)
# Usage: run_suggest_improve input.yaml output.yaml [extra_args...]
run_suggest_improve() {
  local input="$1"
  local output="$2"
  shift 2

  info "Running suggest (improve) ${input} → ${output}"
  run_caster_generator suggest -mapping "$input" -out "$output" "$@"
  stage_done "Improved suggestions saved to ${output}"
}

# Run gen command
# Usage: run_gen mapping.yaml out_dir [extra_args...]
run_gen() {
  local mapping="$1"
  local outdir="$2"
  shift 2

  info "Running gen ${mapping} → ${outdir}"
  clean_dir "$outdir"
  run_caster_generator gen -mapping "$mapping" -out "$outdir" "$@"
  stage_done "Code generated in ${outdir}"
}

# Run gen with write-suggestions (for placeholder transforms)
# Usage: run_gen_with_suggestions mapping.yaml out_dir suggestions.yaml [extra_args...]
run_gen_with_suggestions() {
  local mapping="$1"
  local outdir="$2"
  local suggestions="$3"
  shift 3

  info "Running gen with suggestions ${mapping} → ${outdir}, ${suggestions}"
  clean_dir "$outdir"
  # This may fail if transforms are missing, but will still write suggestions
  run_caster_generator gen -mapping "$mapping" -out "$outdir" -write-suggestions "$suggestions" "$@" || true
  if [[ -f "$suggestions" ]]; then
    stage_done "Suggestions written to ${suggestions}"
  fi
}

# Run check command
# Usage: run_check mapping.yaml
run_check() {
  local mapping="$1"
  shift

  info "Running check ${mapping}"
  run_caster_generator check -mapping "$mapping" "$@"
  stage_done "Mapping validation passed"
}

# Compile-check generated package
# Usage: compile_check package_path
compile_check() {
  local pkg="$1"
  local root
  root="$(repo_root)"

  info "Compile-checking ${pkg}"
  (cd "$root" && go build "${pkg}" 2>&1) || {
    error "Compile check failed for ${pkg}"
    return 1
  }
  stage_done "Package compiles successfully"
}

# Run tests (compile-only, no actual test execution)
# Usage: test_compile package_path
test_compile() {
  local pkg="$1"
  local root
  root="$(repo_root)"

  info "Test-compiling ${pkg}"
  (cd "$root" && go test "${pkg}" -run '^$' -count=1) || {
    error "Test compile failed for ${pkg}"
    return 1
  }
  stage_done "Package test-compiles successfully"
}

# ============================================================================
# Transform Stub Generation
# ============================================================================

# Generate Go stub file for transform functions
# Usage: generate_transform_stubs output.go package_name func_name:signature [...]
# Example: generate_transform_stubs stubs.go mypackage "PriceToInt:func(float64) int64"
generate_transform_stubs() {
  local output="$1"
  local pkg="$2"
  shift 2

  info "Generating transform stubs → ${output}"

  cat > "$output" << EOF
// Code generated by scenario runner. DO NOT EDIT.
// TODO: Implement these transform functions.

package ${pkg}

EOF

  for spec in "$@"; do
    local name="${spec%%:*}"
    local sig="${spec#*:}"

    # Parse signature to extract params and return type
    # Expected format: func(params) return_type
    # or: func(params) (return_type, error)

    cat >> "$output" << EOF
// ${name} transforms source value to target type.
// Signature: ${sig}
// TODO: Implement this function.
var ${name} = ${sig} {
	panic("TODO: implement ${name}")
}

EOF
  done

  stage_done "Transform stubs generated in ${output}"
}

# Extract transform names from YAML file
# Usage: extract_transforms mapping.yaml
extract_transforms() {
  local file="$1"
  grep -E '^\s+transform:\s+\S+' "$file" 2>/dev/null | sed 's/.*transform:\s*//' | sort -u || true
}

# ============================================================================
# Scenario Summary
# ============================================================================

scenario_success() {
  echo ""
  echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║${NC} ${GREEN}✓ Scenario completed successfully!${NC}"
  echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
  echo ""
}

scenario_partial() {
  local msg="${1:-Some steps require manual intervention}"
  echo ""
  echo -e "${YELLOW}╔════════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${YELLOW}║${NC} ${YELLOW}⚠ Scenario partially completed${NC}"
  echo -e "${YELLOW}║${NC} ${msg}"
  echo -e "${YELLOW}╚════════════════════════════════════════════════════════════════╝${NC}"
  echo ""
}

#!/usr/bin/env bash
# Run all tutorial steps sequentially
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║${NC} ${YELLOW}caster-generator Step-by-Step Tutorial${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "This will run all 18 tutorial steps."
echo "Each step demonstrates a feature of caster-generator."
echo ""
echo "Press Ctrl+C at any time to stop."
echo ""

# Optional: skip prompts for CI
if [[ "${1:-}" == "--no-prompt" ]]; then
    export CG_NO_PROMPT=1
fi

for i in {1..18}; do
    step_script="${here}/step${i}.sh"
    if [[ -f "$step_script" ]]; then
        echo ""
        echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${YELLOW}Running Step ${i}${NC}"
        echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
        bash "$step_script"
    else
        echo "Warning: ${step_script} not found, skipping..."
    fi
done

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}All tutorial steps completed!${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════════════════${NC}"

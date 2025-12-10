#!/bin/bash
# Integration test runner for all shells

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

cd "$PROJECT_ROOT"

SHELLS=("bash" "zsh" "fish")
FAILED=0

for shell in "${SHELLS[@]}"; do
    echo ""
    echo -e "${BLUE}=== Testing $shell ===${NC}"

    # Build Docker image
    docker build -t dirvana-test-$shell \
        -f "$SCRIPT_DIR/Dockerfile.$shell" \
        "$PROJECT_ROOT"

    # Run tests
    if docker run --rm dirvana-test-$shell; then
        echo -e "${GREEN}✓ $shell tests passed${NC}"
    else
        echo -e "${RED}✗ $shell tests failed${NC}"
        FAILED=1
    fi

    # Cleanup
    docker rmi dirvana-test-$shell
done

echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}=== All integration tests passed! ===${NC}"
    exit 0
else
    echo -e "${RED}=== Some integration tests failed ===${NC}"
    exit 1
fi

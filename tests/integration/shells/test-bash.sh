#!/usr/local/bin/bash
set -e

echo "=== Testing Dirvana in Bash ==="

# Debug: Check current directory
echo "Current directory: $(pwd)"

# Debug: Check if config exists
echo "Config file exists: $(test -f .dirvana.yml && echo 'YES' || echo 'NO')"

# Debug: Run dirvana export and show output
echo "Running dirvana export..."
SHELL_CODE="$(dirvana export)"
echo "Shell code received:"
echo "$SHELL_CODE"
echo "---"

# Load Dirvana environment
eval "$SHELL_CODE"

# Test 1: Check if alias is loaded
echo ""
echo "Test 1: Checking alias..."
shopt -s expand_aliases  # Enable alias expansion
if alias testcmd &>/dev/null; then
    testcmd
    echo "✓ Alias test passed"
else
    echo "✗ Alias test failed"
    exit 1
fi

# Test 2: Check if function is loaded
echo ""
echo "Test 2: Checking function..."
if declare -f testfunc &>/dev/null; then
    testfunc "parameter"
    echo "✓ Function test passed"
else
    echo "✗ Function test failed"
    exit 1
fi

# Test 3: Check if static env var is loaded
echo ""
echo "Test 3: Checking static environment variable..."
if [ "$TEST_VAR" = "bash-value" ]; then
    echo "TEST_VAR=$TEST_VAR"
    echo "✓ Static env var test passed"
else
    echo "✗ Static env var test failed (got: $TEST_VAR)"
    exit 1
fi

# Test 4: Check if dynamic env var is loaded
echo ""
echo "Test 4: Checking dynamic environment variable..."
if [ "$DYNAMIC_VAR" = "dynamic-bash" ]; then
    echo "DYNAMIC_VAR=$DYNAMIC_VAR"
    echo "✓ Dynamic env var test passed"
else
    echo "✗ Dynamic env var test failed (got: $DYNAMIC_VAR)"
    exit 1
fi

echo ""
echo "=== All Bash tests passed! ==="

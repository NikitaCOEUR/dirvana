#!/usr/bin/zsh
set -e

echo "=== Testing Dirvana in Zsh ==="

# Load Dirvana environment directly
eval "$(dirvana export)"

# Test 1: Check if alias is loaded
echo "Test 1: Checking alias..."
if (( ${+aliases[testcmd]} )); then
    testcmd
    echo "✓ Alias test passed"
else
    echo "✗ Alias test failed"
    exit 1
fi

# Test 2: Check if function is loaded
echo ""
echo "Test 2: Checking function..."
if (( ${+functions[testfunc]} )); then
    testfunc "parameter"
    echo "✓ Function test passed"
else
    echo "✗ Function test failed"
    exit 1
fi

# Test 3: Check if static env var is loaded
echo ""
echo "Test 3: Checking static environment variable..."
if [[ "$TEST_VAR" == "zsh-value" ]]; then
    echo "TEST_VAR=$TEST_VAR"
    echo "✓ Static env var test passed"
else
    echo "✗ Static env var test failed (got: $TEST_VAR)"
    exit 1
fi

# Test 4: Check if dynamic env var is loaded
echo ""
echo "Test 4: Checking dynamic environment variable..."
if [[ "$DYNAMIC_VAR" == "dynamic-zsh" ]]; then
    echo "DYNAMIC_VAR=$DYNAMIC_VAR"
    echo "✓ Dynamic env var test passed"
else
    echo "✗ Dynamic env var test failed (got: $DYNAMIC_VAR)"
    exit 1
fi

echo ""
echo "=== All Zsh tests passed! ==="

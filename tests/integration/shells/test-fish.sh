#!/usr/bin/env fish
# Integration tests for Dirvana in Fish shell

echo "=== Testing Dirvana in Fish ==="
echo ""

# Debug: Check current directory
echo "Current directory: "(pwd)
echo "Config file exists: "(test -f .dirvana.yml; and echo 'YES'; or echo 'NO')
echo ""

# Load Dirvana environment
echo "Loading Dirvana environment..."
set shell_code (dirvana export)
if test $status -ne 0
    echo "✗ Failed to run dirvana export"
    exit 1
end

# Fish requires sourcing from a file
set tmp_file (mktemp)
printf '%s\n' $shell_code > $tmp_file
source $tmp_file
rm -f $tmp_file
echo "✓ Environment loaded"
echo ""

# Test 1: Simple alias
echo "Test 1: Simple alias..."
if functions -q testcmd
    set output (testcmd)
    if string match -q "*Dirvana alias works in fish*" $output
        echo "✓ Simple alias works: $output"
    else
        echo "✗ Simple alias failed: $output"
        exit 1
    end
else
    echo "✗ Alias not loaded"
    exit 1
end
echo ""

# Test 2: Alias with options
echo "Test 2: Alias with options (ll)..."
if functions -q ll
    echo "✓ Complex alias 'll' is loaded"
else
    echo "✗ Complex alias failed"
    exit 1
end
echo ""

# Test 3: Simple function
echo "Test 3: Simple function..."
if functions -q testfunc
    set output (testfunc "test-param")
    if string match -q "*Dirvana function works: test-param*" $output
        echo "✓ Simple function works: $output"
    else
        echo "✗ Simple function failed: $output"
        exit 1
    end
else
    echo "✗ Function not loaded"
    exit 1
end
echo ""

# Test 4: Function with logic
echo "Test 4: Function with conditionals (greet)..."
if functions -q greet
    set output1 (greet)
    set output2 (greet "World")
    if string match -q "*stranger*" $output1; and string match -q "*World*" $output2
        echo "✓ Conditional function works"
        echo "  Without param: $output1"
        echo "  With param: $output2"
    else
        echo "✗ Conditional function failed"
        echo "  Without param: $output1"
        echo "  With param: $output2"
        exit 1
    end
else
    echo "✗ greet function not loaded"
    exit 1
end
echo ""

# Test 5: Environment variables
echo "Test 5: Environment variables..."
if test -n "$PROJECT_NAME"; and test -n "$ENVIRONMENT"
    echo "✓ Environment variables loaded"
    echo "  PROJECT_NAME=$PROJECT_NAME"
    echo "  ENVIRONMENT=$ENVIRONMENT"
else
    echo "✗ Environment variables not loaded"
    exit 1
end
echo ""

# Test 6: Dynamic environment variables
echo "Test 6: Dynamic environment variables (from sh: commands)..."
if test -n "$CURRENT_USER"
    echo "✓ Dynamic env vars work"
    echo "  CURRENT_USER=$CURRENT_USER"
else
    echo "✗ Dynamic env vars failed"
    exit 1
end
echo ""

# Test 7: Function using environment variables
echo "Test 7: Function using environment variables..."
if functions -q showenv
    set output (showenv)
    if string match -q "*Project: dirvana-test*" $output
        echo "✓ Function can access environment variables"
        echo "$output" | string split \n | string replace -r '^' '  '
    else
        echo "✗ Function couldn't access env vars"
        echo "$output"
        exit 1
    end
else
    echo "✗ showenv function not loaded"
    exit 1
end
echo ""

# Test 8: Alias with arguments (critical for Fish)
echo "Test 8: Alias with arguments (critical for Fish)..."
if functions -q testcmd
    set output (testcmd arg1 arg2)
    if test $status -eq 0
        echo "✓ Alias with arguments works"
    else
        echo "✗ Alias with arguments failed with status: $status"
        exit 1
    end
else
    echo "✗ testcmd not loaded"
    exit 1
end
echo ""

echo "=== All Fish tests passed! ==="
exit 0

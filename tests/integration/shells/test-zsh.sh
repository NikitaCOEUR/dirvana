#!/usr/bin/zsh
set -e

echo "=== Testing Dirvana in Zsh ==="
echo ""

# Debug: Check current directory
echo "Current directory: $(pwd)"
echo "Config file exists: $(test -f .dirvana.yml && echo 'YES' || echo 'NO')"
echo ""

# Load Dirvana environment
echo "Loading Dirvana environment..."
eval "$(dirvana export)"
echo "✓ Environment loaded"
echo ""

# Test 1: Simple alias
echo "Test 1: Simple alias..."
if (( ${+aliases[testcmd]} )); then
    OUTPUT=$(testcmd)
    if [[ "$OUTPUT" == *"Dirvana alias works in zsh"* ]]; then
        echo "✓ Simple alias works: $OUTPUT"
    else
        echo "✗ Simple alias failed"
        exit 1
    fi
else
    echo "✗ Alias not loaded"
    exit 1
fi
echo ""

# Test 2: Alias with options
echo "Test 2: Alias with options (ll)..."
if (( ${+aliases[ll]} )); then
    echo "✓ Complex alias 'll' is loaded"
else
    echo "✗ Complex alias failed"
    exit 1
fi
echo ""

# Test 3: Simple function
echo "Test 3: Simple function..."
if (( ${+functions[testfunc]} )); then
    OUTPUT=$(testfunc "test-param")
    if [[ "$OUTPUT" == *"Dirvana function works: test-param"* ]]; then
        echo "✓ Simple function works: $OUTPUT"
    else
        echo "✗ Simple function failed: $OUTPUT"
        exit 1
    fi
else
    echo "✗ Function not loaded"
    exit 1
fi
echo ""

# Test 4: Function with logic
echo "Test 4: Function with conditionals (greet)..."
if (( ${+functions[greet]} )); then
    OUTPUT1=$(greet)
    OUTPUT2=$(greet "World")
    if [[ "$OUTPUT1" == *"stranger"* ]] && [[ "$OUTPUT2" == *"World"* ]]; then
        echo "✓ Conditional function works"
        echo "  Without param: $OUTPUT1"
        echo "  With param: $OUTPUT2"
    else
        echo "✗ Conditional function failed"
        exit 1
    fi
else
    echo "✗ greet function not loaded"
    exit 1
fi
echo ""

# Test 5: Function with arrays
echo "Test 5: Function with arrays (listargs)..."
if (( ${+functions[listargs]} )); then
    OUTPUT=$(listargs "arg1" "arg2" "arg3")
    if [[ "$OUTPUT" == *"Arg 1: arg1"* ]] && [[ "$OUTPUT" == *"Arg 3: arg3"* ]]; then
        echo "✓ Array function works"
        echo "$OUTPUT" | sed 's/^/  /'
    else
        echo "✗ Array function failed"
        exit 1
    fi
else
    echo "✗ listargs function not loaded"
    exit 1
fi
echo ""

# Test 6: Function with file check
echo "Test 6: Function with file conditionals (checkfile)..."
if (( ${+functions[checkfile]} )); then
    # Create a test file
    touch /tmp/test-file.txt
    OUTPUT=$(checkfile /tmp/test-file.txt)
    rm -f /tmp/test-file.txt

    if [[ "$OUTPUT" == *"File exists"* ]]; then
        echo "✓ File check function works"
    else
        echo "✗ File check function failed"
        exit 1
    fi
else
    echo "✗ checkfile function not loaded"
    exit 1
fi
echo ""

# Test 7: Static environment variables
echo "Test 7: Static environment variables..."
PASSED=0
FAILED=0

if [[ "$PROJECT_NAME" == "dirvana-test" ]]; then
    echo "✓ PROJECT_NAME=$PROJECT_NAME"
    PASSED=$((PASSED + 1))
else
    echo "✗ PROJECT_NAME failed (got: $PROJECT_NAME)"
    FAILED=$((FAILED + 1))
fi

if [[ "$ENVIRONMENT" == "integration" ]]; then
    echo "✓ ENVIRONMENT=$ENVIRONMENT"
    PASSED=$((PASSED + 1))
else
    echo "✗ ENVIRONMENT failed (got: $ENVIRONMENT)"
    FAILED=$((FAILED + 1))
fi

if [[ "$DEBUG" == "true" ]]; then
    echo "✓ DEBUG=$DEBUG"
    PASSED=$((PASSED + 1))
else
    echo "✗ DEBUG failed (got: $DEBUG)"
    FAILED=$((FAILED + 1))
fi

if [[ "$MAX_WORKERS" == "4" ]]; then
    echo "✓ MAX_WORKERS=$MAX_WORKERS"
    PASSED=$((PASSED + 1))
else
    echo "✗ MAX_WORKERS failed (got: $MAX_WORKERS)"
    FAILED=$((FAILED + 1))
fi

if [[ "$RETRY_COUNT" == "3" ]]; then
    echo "✓ RETRY_COUNT=$RETRY_COUNT"
    PASSED=$((PASSED + 1))
else
    echo "✗ RETRY_COUNT failed (got: $RETRY_COUNT)"
    FAILED=$((FAILED + 1))
fi

echo "Static vars: $PASSED passed, $FAILED failed"
if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
echo ""

# Test 8: Dynamic environment variables
echo "Test 8: Dynamic environment variables..."
PASSED=0
FAILED=0

if [[ -n "$CURRENT_USER" ]]; then
    echo "✓ CURRENT_USER=$CURRENT_USER (dynamic)"
    PASSED=$((PASSED + 1))
else
    echo "✗ CURRENT_USER not set"
    FAILED=$((FAILED + 1))
fi

if [[ "$CURRENT_DIR" == "/test/project" ]]; then
    echo "✓ CURRENT_DIR=$CURRENT_DIR (dynamic)"
    PASSED=$((PASSED + 1))
else
    echo "✗ CURRENT_DIR failed (got: $CURRENT_DIR)"
    FAILED=$((FAILED + 1))
fi

if [[ -n "$TIMESTAMP" ]] && [[ "$TIMESTAMP" =~ ^[0-9]+$ ]]; then
    echo "✓ TIMESTAMP=$TIMESTAMP (dynamic)"
    PASSED=$((PASSED + 1))
else
    echo "✗ TIMESTAMP failed (got: $TIMESTAMP)"
    FAILED=$((FAILED + 1))
fi

if [[ -n "$GIT_BRANCH" ]]; then
    echo "✓ GIT_BRANCH=$GIT_BRANCH (dynamic)"
    PASSED=$((PASSED + 1))
else
    echo "✗ GIT_BRANCH not set"
    FAILED=$((FAILED + 1))
fi

if [[ -n "$HOSTNAME" ]]; then
    echo "✓ HOSTNAME=$HOSTNAME (dynamic)"
    PASSED=$((PASSED + 1))
else
    echo "✗ HOSTNAME not set"
    FAILED=$((FAILED + 1))
fi

echo "Dynamic vars: $PASSED passed, $FAILED failed"
if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
echo ""

# Test 9: Function using environment variables
echo "Test 9: Function using environment variables..."
if (( ${+functions[showenv]} )); then
    OUTPUT=$(showenv)
    if [[ "$OUTPUT" == *"$PROJECT_NAME"* ]] && [[ "$OUTPUT" == *"$ENVIRONMENT"* ]]; then
        echo "✓ Function can access env vars"
        echo "$OUTPUT" | sed 's/^/  /'
    else
        echo "✗ Function env var access failed"
        exit 1
    fi
else
    echo "✗ showenv function not loaded"
    exit 1
fi
echo ""

# Test 10: Path variables
echo "Test 10: Path environment variables..."
if [[ "$BUILD_DIR" == "/tmp/build" ]] && [[ "$CACHE_DIR" == "/tmp/cache" ]] && [[ "$OUTPUT_DIR" == "/tmp/output" ]]; then
    echo "✓ Path variables set correctly"
    echo "  BUILD_DIR=$BUILD_DIR"
    echo "  CACHE_DIR=$CACHE_DIR"
    echo "  OUTPUT_DIR=$OUTPUT_DIR"
else
    echo "✗ Path variables failed"
    exit 1
fi
echo ""

echo "================================================"
echo "=== ✓ All Zsh tests passed successfully! ==="
echo "================================================"
echo ""
echo "Summary:"
echo "- Aliases: simple, complex (with options)"
echo "- Functions: simple, conditionals, arrays, file checks, env var access"
echo "- Env vars: static, dynamic (shell commands), paths, numeric"
echo "- Config flags: local_only, ignore_global"

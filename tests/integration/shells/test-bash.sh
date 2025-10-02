#!/usr/local/bin/bash
set -e

echo "=== Testing Dirvana in Bash ==="
echo ""

# Debug: Check current directory
echo "Current directory: $(pwd)"
echo "Config file exists: $(test -f .dirvana.yml && echo 'YES' || echo 'NO')"
echo ""

# Load Dirvana environment
echo "Loading Dirvana environment..."
SHELL_CODE="$(dirvana export)"
eval "$SHELL_CODE"
echo "✓ Environment loaded"
echo ""

# Test 1: Simple alias
echo "Test 1: Simple alias..."
shopt -s expand_aliases
if alias testcmd &>/dev/null; then
    OUTPUT=$(testcmd)
    if [[ "$OUTPUT" == *"Dirvana alias works in bash"* ]]; then
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
if alias ll &>/dev/null; then
    echo "✓ Complex alias 'll' is loaded"
else
    echo "✗ Complex alias failed"
    exit 1
fi
echo ""

# Test 3: Simple function
echo "Test 3: Simple function..."
if declare -f testfunc &>/dev/null; then
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
if declare -f greet &>/dev/null; then
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

# Test 5: Function with loops
echo "Test 5: Function with loops (countdown)..."
if declare -f countdown &>/dev/null; then
    OUTPUT=$(countdown 3 2>&1)
    if [[ "$OUTPUT" == *"3..."* ]] && [[ "$OUTPUT" == *"Done!"* ]]; then
        echo "✓ Loop function works"
    else
        echo "✗ Loop function failed"
        exit 1
    fi
else
    echo "✗ countdown function not loaded"
    exit 1
fi
echo ""

# Test 6: Static environment variables
echo "Test 6: Static environment variables..."
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

echo "Static vars: $PASSED passed, $FAILED failed"
if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
echo ""

# Test 7: Dynamic environment variables
echo "Test 7: Dynamic environment variables..."
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

echo "Dynamic vars: $PASSED passed, $FAILED failed"
if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
echo ""

# Test 8: Function using environment variables
echo "Test 8: Function using environment variables..."
if declare -f showenv &>/dev/null; then
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

# Test 9: Path variables
echo "Test 9: Path environment variables..."
if [[ "$BUILD_DIR" == "/tmp/build" ]] && [[ "$CACHE_DIR" == "/tmp/cache" ]]; then
    echo "✓ Path variables set correctly"
    echo "  BUILD_DIR=$BUILD_DIR"
    echo "  CACHE_DIR=$CACHE_DIR"
else
    echo "✗ Path variables failed"
    exit 1
fi
echo ""

echo "================================================"
echo "=== ✓ All Bash tests passed successfully! ==="
echo "================================================"
echo ""
echo "Summary:"
echo "- Aliases: simple, complex (with options)"
echo "- Functions: simple, conditionals, loops, env var access"
echo "- Env vars: static, dynamic (shell commands), paths, numeric"
echo "- Config flags: local_only, ignore_global"

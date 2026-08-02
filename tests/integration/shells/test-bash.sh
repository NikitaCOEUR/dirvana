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

if [[ "$LOG_LEVEL" == "debug" ]]; then
    echo "✓ LOG_LEVEL=$LOG_LEVEL"
    PASSED=$((PASSED + 1))
else
    echo "✗ LOG_LEVEL failed (got: $LOG_LEVEL)"
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

# Test 10: Alias with arguments (forwarded through dirvana exec)
echo "Test 10: Alias with arguments..."
OUTPUT=$(testcmd extra-arg)
if [[ "$OUTPUT" == *"Dirvana alias works in bash"* ]] && [[ "$OUTPUT" == *"extra-arg"* ]]; then
    echo "✓ Alias forwards arguments: $OUTPUT"
else
    echo "✗ Alias arguments failed (got: $OUTPUT)"
    exit 1
fi
echo ""

# Test 11: Conditional alias (when/else evaluated at execution time)
echo "Test 11: Conditional alias (when/else)..."
rm -f marker.txt
OUTPUT=$(condcmd)
if [[ "$OUTPUT" == *"condition-missing"* ]]; then
    echo "✓ else branch when condition unmet: $OUTPUT"
else
    echo "✗ else branch failed (got: $OUTPUT)"
    exit 1
fi
touch marker.txt
OUTPUT=$(condcmd)
if [[ "$OUTPUT" == *"condition-met"* ]]; then
    echo "✓ main command when condition met: $OUTPUT"
else
    echo "✗ when branch failed (got: $OUTPUT)"
    exit 1
fi
rm -f marker.txt
echo ""

# Test 12: Hierarchy - child inherits and overrides parent
echo "Test 12: Config hierarchy (child dir)..."
mkdir -p /test/project/sub
cat > /test/project/sub/.dirvana.yml <<'EOF'
aliases:
  testcmd: echo "overridden by child"
  subonly: echo "child only alias"
env:
  SUB_VAR: sub-value
EOF
dirvana allow /test/project/sub >/dev/null
cd /test/project/sub
# The inherited env sh: commands need per-directory approval on first entry;
# DIRVANA_TEST_MODE routes the prompt to stdin so we can approve with "y"
eval "$(echo y | DIRVANA_TEST_MODE=1 dirvana export --prev /test/project)"
OUTPUT=$(testcmd)
if [[ "$OUTPUT" == *"overridden by child"* ]]; then
    echo "✓ Child overrides parent alias"
else
    echo "✗ Child override failed (got: $OUTPUT)"
    exit 1
fi
OUTPUT=$(subonly)
if [[ "$OUTPUT" == *"child only alias"* ]] && [[ "$SUB_VAR" == "sub-value" ]]; then
    echo "✓ Child-only alias and env var defined"
else
    echo "✗ Child additions failed"
    exit 1
fi
if [[ "$PROJECT_NAME" == "dirvana-test" ]]; then
    echo "✓ Parent env var inherited in child dir"
else
    echo "✗ Parent inheritance failed (got: $PROJECT_NAME)"
    exit 1
fi
echo ""

# Test 13: Hierarchy - going back to parent cleans child definitions
echo "Test 13: Cleanup when returning to parent..."
cd /test/project
eval "$(dirvana export --prev /test/project/sub)"
if alias subonly &>/dev/null; then
    echo "✗ Child-only alias survived leaving the child dir"
    exit 1
fi
if [[ -n "$SUB_VAR" ]]; then
    echo "✗ Child env var survived leaving the child dir (got: $SUB_VAR)"
    exit 1
fi
OUTPUT=$(testcmd)
if [[ "$OUTPUT" == *"Dirvana alias works in bash"* ]]; then
    echo "✓ Child definitions cleaned, parent alias restored"
else
    echo "✗ Parent alias not restored (got: $OUTPUT)"
    exit 1
fi
echo ""

# Test 14: local_only ignores parent configs
echo "Test 14: local_only flag..."
mkdir -p /test/project/sublocal
cat > /test/project/sublocal/.dirvana.yml <<'EOF'
local_only: true
aliases:
  localcmd: echo "local only context"
EOF
dirvana allow /test/project/sublocal >/dev/null
cd /test/project/sublocal
eval "$(dirvana export --prev /test/project)"
OUTPUT=$(localcmd)
if [[ "$OUTPUT" != *"local only context"* ]]; then
    echo "✗ local_only alias failed (got: $OUTPUT)"
    exit 1
fi
if alias testcmd &>/dev/null; then
    echo "✗ Parent alias leaked into local_only context"
    exit 1
fi
if [[ -n "$PROJECT_NAME" ]]; then
    echo "✗ Parent env var leaked into local_only context (got: $PROJECT_NAME)"
    exit 1
fi
echo "✓ local_only isolates from parent configs"
echo ""

# Test 15: global config and ignore_global
echo "Test 15: Global config and ignore_global..."
mkdir -p ~/.config/dirvana
cat > ~/.config/dirvana/global.yml <<'EOF'
aliases:
  globalcmd: echo "from global config"
EOF
cd /test/project
SHELL_CODE=$(dirvana export --prev /test/project/sublocal)
if [[ "$SHELL_CODE" == *"globalcmd"* ]]; then
    echo "✓ Global config merged into project"
else
    echo "✗ Global config not merged"
    exit 1
fi
eval "$SHELL_CODE"
mkdir -p /test/project/subglobal
cat > /test/project/subglobal/.dirvana.yml <<'EOF'
ignore_global: true
aliases:
  sgcmd: echo "subglobal"
EOF
dirvana allow /test/project/subglobal >/dev/null
cd /test/project/subglobal
SHELL_CODE=$(echo y | DIRVANA_TEST_MODE=1 dirvana export --prev /test/project)
if [[ "$SHELL_CODE" == *"globalcmd"* ]]; then
    echo "✗ ignore_global still emitted the global alias"
    exit 1
fi
if [[ "$SHELL_CODE" != *"sgcmd"* ]] || [[ "$SHELL_CODE" != *"testcmd"* ]]; then
    echo "✗ ignore_global dropped local hierarchy definitions"
    exit 1
fi
echo "✓ ignore_global skips the global config but keeps the hierarchy"
eval "$SHELL_CODE"
rm -f ~/.config/dirvana/global.yml
echo ""

# Test 16: Unauthorized directory must not load its config
echo "Test 16: Unauthorized directory..."
mkdir -p /test/unauth
cat > /test/unauth/.dirvana.yml <<'EOF'
aliases:
  evilcmd: echo "should never be defined"
EOF
cd /test/unauth
# The guidance message must reach the user even though the hook discards
# stderr: it goes through /dev/tty (or stderr in test mode)
WARN_OUTPUT=$(DIRVANA_TEST_MODE=1 dirvana export --prev /test/project/subglobal 2>&1 >/dev/null)
if [[ "$WARN_OUTPUT" != *"not authorized"* ]]; then
    echo "✗ Unauthorized-config notice not shown to the user"
    exit 1
fi
echo "✓ Unauthorized-config notice is visible"
SHELL_CODE=$(dirvana export --prev /test/project/subglobal 2>/dev/null)
if [[ "$SHELL_CODE" == *"evilcmd"* ]]; then
    echo "✗ SECURITY: unauthorized config was loaded!"
    exit 1
fi
echo "✓ Unauthorized config not loaded"
eval "$SHELL_CODE"
echo ""

# Test 17: Cleanup when leaving to an unconfigured context
echo "Test 17: Full cleanup after leaving dirvana context..."
if alias testcmd &>/dev/null; then
    echo "✗ Alias survived leaving the context"
    exit 1
fi
if declare -f testfunc &>/dev/null; then
    echo "✗ Function survived leaving the context"
    exit 1
fi
if [[ -n "$PROJECT_NAME" ]]; then
    echo "✗ Env var survived leaving the context (got: $PROJECT_NAME)"
    exit 1
fi
echo "✓ Aliases, functions and env vars cleaned up"
echo ""

echo "================================================"
echo "=== ✓ All Bash tests passed successfully! ==="
echo "================================================"
echo ""
echo "Summary:"
echo "- Aliases: simple, complex, with arguments, conditional (when/else)"
echo "- Functions: simple, conditionals, loops, env var access"
echo "- Env vars: static, dynamic (shell commands), paths, numeric"
echo "- Hierarchy: inheritance, override, cleanup on directory change"
echo "- Config flags: local_only, ignore_global (actually exercised)"
echo "- Security: unauthorized directory config is not loaded"

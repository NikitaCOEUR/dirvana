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

# Helper: evaluate `dirvana export` output (fish must source from a file).
# Usage: load_dirvana <prev-dir> [approve]
function load_dirvana
    set -l prev $argv[1]
    if test "$argv[2]" = approve
        set shell_code (echo y | env DIRVANA_TEST_MODE=1 dirvana export --prev $prev)
    else
        set shell_code (dirvana export --prev $prev 2>/dev/null)
    end
    set -l tmp_file (mktemp)
    printf '%s\n' $shell_code > $tmp_file
    source $tmp_file
    rm -f $tmp_file
end

# Test 9: Conditional alias (when/else evaluated at execution time)
echo "Test 9: Conditional alias (when/else)..."
rm -f marker.txt
set output (condcmd)
if string match -q "*condition-missing*" $output
    echo "✓ else branch when condition unmet: $output"
else
    echo "✗ else branch failed (got: $output)"
    exit 1
end
touch marker.txt
set output (condcmd)
if string match -q "*condition-met*" $output
    echo "✓ main command when condition met: $output"
else
    echo "✗ when branch failed (got: $output)"
    exit 1
end
rm -f marker.txt
echo ""

# Test 10: Hierarchy - child inherits and overrides parent
echo "Test 10: Config hierarchy (child dir)..."
mkdir -p /test/project/sub
printf '%s\n' \
    'aliases:' \
    '  testcmd: echo "overridden by child"' \
    '  subonly: echo "child only alias"' \
    'env:' \
    '  SUB_VAR: sub-value' \
    > /test/project/sub/.dirvana.yml
dirvana allow /test/project/sub >/dev/null
cd /test/project/sub
# The inherited env sh: commands need per-directory approval on first entry
load_dirvana /test/project approve
set output (testcmd)
if not string match -q "*overridden by child*" $output
    echo "✗ Child override failed (got: $output)"
    exit 1
end
echo "✓ Child overrides parent alias"
set output (subonly)
if string match -q "*child only alias*" $output; and test "$SUB_VAR" = "sub-value"
    echo "✓ Child-only alias and env var defined"
else
    echo "✗ Child additions failed"
    exit 1
end
if test "$PROJECT_NAME" = "dirvana-test"
    echo "✓ Parent env var inherited in child dir"
else
    echo "✗ Parent inheritance failed (got: $PROJECT_NAME)"
    exit 1
end
echo ""

# Test 11: Hierarchy - going back to parent cleans child definitions
echo "Test 11: Cleanup when returning to parent..."
cd /test/project
load_dirvana /test/project/sub
if functions -q subonly
    echo "✗ Child-only alias survived leaving the child dir"
    exit 1
end
if set -q SUB_VAR
    echo "✗ Child env var survived leaving the child dir (got: $SUB_VAR)"
    exit 1
end
set output (testcmd)
if string match -q "*Dirvana alias works in fish*" $output
    echo "✓ Child definitions cleaned, parent alias restored"
else
    echo "✗ Parent alias not restored (got: $output)"
    exit 1
end
echo ""

# Test 12: local_only ignores parent configs
echo "Test 12: local_only flag..."
mkdir -p /test/project/sublocal
printf '%s\n' \
    'local_only: true' \
    'aliases:' \
    '  localcmd: echo "local only context"' \
    > /test/project/sublocal/.dirvana.yml
dirvana allow /test/project/sublocal >/dev/null
cd /test/project/sublocal
load_dirvana /test/project
set output (localcmd)
if not string match -q "*local only context*" $output
    echo "✗ local_only alias failed (got: $output)"
    exit 1
end
if functions -q testcmd
    echo "✗ Parent alias leaked into local_only context"
    exit 1
end
if set -q PROJECT_NAME
    echo "✗ Parent env var leaked into local_only context (got: $PROJECT_NAME)"
    exit 1
end
echo "✓ local_only isolates from parent configs"
echo ""

# Test 13: global config and ignore_global
echo "Test 13: Global config and ignore_global..."
mkdir -p ~/.config/dirvana
printf '%s\n' \
    'aliases:' \
    '  globalcmd: echo "from global config"' \
    > ~/.config/dirvana/global.yml
cd /test/project
set shell_code (dirvana export --prev /test/project/sublocal)
if not string match -q "*globalcmd*" "$shell_code"
    echo "✗ Global config not merged"
    exit 1
end
echo "✓ Global config merged into project"
set tmp_file (mktemp)
printf '%s\n' $shell_code > $tmp_file
source $tmp_file
rm -f $tmp_file
mkdir -p /test/project/subglobal
printf '%s\n' \
    'ignore_global: true' \
    'aliases:' \
    '  sgcmd: echo "subglobal"' \
    > /test/project/subglobal/.dirvana.yml
dirvana allow /test/project/subglobal >/dev/null
cd /test/project/subglobal
set shell_code (echo y | env DIRVANA_TEST_MODE=1 dirvana export --prev /test/project)
if string match -q "*globalcmd*" "$shell_code"
    echo "✗ ignore_global still emitted the global alias"
    exit 1
end
if not string match -q "*sgcmd*" "$shell_code"; or not string match -q "*testcmd*" "$shell_code"
    echo "✗ ignore_global dropped local hierarchy definitions"
    exit 1
end
echo "✓ ignore_global skips the global config but keeps the hierarchy"
set tmp_file (mktemp)
printf '%s\n' $shell_code > $tmp_file
source $tmp_file
rm -f $tmp_file
rm -f ~/.config/dirvana/global.yml
echo ""

# Test 14: Unauthorized directory must not load its config
echo "Test 14: Unauthorized directory..."
mkdir -p /test/unauth
printf '%s\n' \
    'aliases:' \
    '  evilcmd: echo "should never be defined"' \
    > /test/unauth/.dirvana.yml
cd /test/unauth
set shell_code (dirvana export --prev /test/project/subglobal 2>/dev/null)
if string match -q "*evilcmd*" "$shell_code"
    echo "✗ SECURITY: unauthorized config was loaded!"
    exit 1
end
echo "✓ Unauthorized config not loaded"
set tmp_file (mktemp)
printf '%s\n' $shell_code > $tmp_file
source $tmp_file
rm -f $tmp_file
echo ""

# Test 15: Cleanup when leaving to an unconfigured context
echo "Test 15: Full cleanup after leaving dirvana context..."
if functions -q testcmd
    echo "✗ Alias survived leaving the context"
    exit 1
end
if functions -q testfunc
    echo "✗ Function survived leaving the context"
    exit 1
end
if set -q PROJECT_NAME
    echo "✗ Env var survived leaving the context (got: $PROJECT_NAME)"
    exit 1
end
echo "✓ Aliases, functions and env vars cleaned up"
echo ""

echo "=== All Fish tests passed! ==="
echo ""
echo "Summary:"
echo "- Aliases: simple, complex, with arguments, conditional (when/else)"
echo "- Functions: simple, conditionals, env var access"
echo "- Env vars: static, dynamic (shell commands)"
echo "- Hierarchy: inheritance, override, cleanup on directory change"
echo "- Config flags: local_only, ignore_global (actually exercised)"
echo "- Security: unauthorized directory config is not loaded"
exit 0

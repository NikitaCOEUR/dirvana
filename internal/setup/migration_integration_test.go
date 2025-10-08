package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLegacyToExternalMigration tests the complete migration process
// from legacy inline hook to external hook strategy
func TestLegacyToExternalMigration(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()
	require.NoError(t, os.Setenv("HOME", tmpDir))

	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Step 1: Simulate legacy installation
	legacyHookCode := `__dirvana_hook() {
  # Old legacy hook code
  echo "legacy"
}

if [[ -z "${PROMPT_COMMAND}" ]]; then
  PROMPT_COMMAND="__dirvana_hook"
else
  PROMPT_COMMAND="__dirvana_hook;${PROMPT_COMMAND}"
fi`

	legacyContent := fmt.Sprintf(`# User's custom bashrc
export PATH=$PATH:/usr/local/bin
alias ll='ls -la'

%s
%s
%s

# More user content
export EDITOR=vim
`, HookMarkerStart, legacyHookCode, HookMarkerEnd)

	require.NoError(t, os.WriteFile(rcFile, []byte(legacyContent), 0644))

	// Step 2: Verify legacy installation is detected
	hasLegacy := HasLegacyInstall("bash")
	assert.True(t, hasLegacy, "Should detect legacy installation")

	// Step 3: Run setup (should trigger migration)
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated, "Should report as updated (migrated)")
	assert.Contains(t, result.Message, "Migrated", "Message should mention migration")

	// Step 4: Verify legacy markers are removed from RC file
	rcContent, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	rcStr := string(rcContent)

	assert.NotContains(t, rcStr, HookMarkerStart, "Legacy start marker should be removed")
	assert.NotContains(t, rcStr, HookMarkerEnd, "Legacy end marker should be removed")
	assert.NotContains(t, rcStr, "# Old legacy hook code", "Legacy hook code should be removed")
	assert.NotContains(t, rcStr, `echo "legacy"`, "Legacy hook code should be removed")

	// Step 5: Verify user content is preserved
	assert.Contains(t, rcStr, "# User's custom bashrc", "User content should be preserved")
	assert.Contains(t, rcStr, "export PATH=$PATH:/usr/local/bin", "User content should be preserved")
	assert.Contains(t, rcStr, "alias ll='ls -la'", "User content should be preserved")
	assert.Contains(t, rcStr, "# More user content", "User content should be preserved")
	assert.Contains(t, rcStr, "export EDITOR=vim", "User content should be preserved")

	// Step 6: Verify new external hook file was created
	configDir := filepath.Join(tmpDir, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")

	_, err = os.Stat(hookPath)
	assert.NoError(t, err, "External hook file should exist")

	// Step 7: Verify hook file contains current hook code
	hookContent, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	expectedHook := cli.GenerateHookCode("bash")
	assert.Equal(t, expectedHook, string(hookContent), "Hook file should contain current hook code")

	// Step 8: Verify RC file now sources the external hook
	assert.Contains(t, rcStr, hookPath, "RC file should reference external hook file")
	assert.Contains(t, rcStr, DirvanaComment, "RC file should have Dirvana comment")

	// Step 9: Verify no legacy markers remain
	assert.False(t, HasLegacyInstall("bash"), "Legacy installation should no longer be detected")

	// Step 10: Verify subsequent setup doesn't re-migrate
	result2, err := InstallHook("bash")
	require.NoError(t, err)
	assert.False(t, result2.Updated, "Second setup should not report as updated")
	assert.NotContains(t, result2.Message, "Migrated", "Should not mention migration again")
	assert.Contains(t, result2.Message, "up to date", "Should report as up to date")
}

// TestLegacyUninstall tests uninstalling a legacy installation
func TestLegacyUninstall(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()
	require.NoError(t, os.Setenv("HOME", tmpDir))

	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create legacy installation
	legacyContent := fmt.Sprintf(`# User content
export PATH=$PATH:/usr/local/bin

%s
# Legacy hook
%s

# More user content
`, HookMarkerStart, HookMarkerEnd)

	require.NoError(t, os.WriteFile(rcFile, []byte(legacyContent), 0644))

	// Verify legacy is detected
	assert.True(t, HasLegacyInstall("bash"), "Should detect legacy")

	// Uninstall
	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated, "Should report as updated")
	assert.True(t, strings.Contains(result.Message, "removed") || strings.Contains(result.Message, "Removed"), "Should mention removal")

	// Verify legacy markers are gone
	rcContent, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	rcStr := string(rcContent)

	assert.NotContains(t, rcStr, HookMarkerStart)
	assert.NotContains(t, rcStr, HookMarkerEnd)
	assert.NotContains(t, rcStr, "# Legacy hook")

	// Verify user content is preserved
	assert.Contains(t, rcStr, "# User content")
	assert.Contains(t, rcStr, "export PATH=$PATH:/usr/local/bin")
	assert.Contains(t, rcStr, "# More user content")

	// Verify no longer detected as legacy
	assert.False(t, HasLegacyInstall("bash"))
}

// TestMigrationPreservesComplexContent tests that migration preserves
// complex user content including functions, loops, conditionals, etc.
func TestMigrationPreservesComplexContent(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()
	require.NoError(t, os.Setenv("HOME", tmpDir))

	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Complex user content with various shell constructs
	complexContent := `# Complex bashrc
export PATH=$PATH:/usr/local/bin

# Function definitions
function git_branch() {
  git branch 2>/dev/null | grep '*' | cut -d ' ' -f2
}

# Conditional logic
if [ -f ~/.bash_aliases ]; then
  source ~/.bash_aliases
fi

# Loop example
for dir in ~/bin/*; do
  [ -d "$dir" ] && PATH="$PATH:$dir"
done

# Array
declare -a my_array=("one" "two" "three")

` + fmt.Sprintf(`%s
# Legacy hook
%s

`, HookMarkerStart, HookMarkerEnd) + `# More complex content after hook
case "$TERM" in
  xterm*|rxvt*)
    PS1='\[\e]0;\u@\h: \w\a\]${debian_chroot:+($debian_chroot)}\u@\h:\w\$ '
    ;;
  *)
    PS1='${debian_chroot:+($debian_chroot)}\u@\h:\w\$ '
    ;;
esac

# Aliases with special characters
alias grep='grep --color=auto'
alias ls='ls --color=auto'
`

	require.NoError(t, os.WriteFile(rcFile, []byte(complexContent), 0644))

	// Migrate
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	// Verify all complex content is preserved
	rcContent, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	rcStr := string(rcContent)

	// Check all the complex constructs are still there
	assert.Contains(t, rcStr, "function git_branch()")
	assert.Contains(t, rcStr, "if [ -f ~/.bash_aliases ]")
	assert.Contains(t, rcStr, "for dir in ~/bin/*")
	assert.Contains(t, rcStr, "declare -a my_array")
	assert.Contains(t, rcStr, `case "$TERM" in`)
	assert.Contains(t, rcStr, "alias grep='grep --color=auto'")

	// Verify legacy hook is gone
	assert.NotContains(t, rcStr, HookMarkerStart)
	assert.NotContains(t, rcStr, HookMarkerEnd)
	assert.NotContains(t, rcStr, "# Legacy hook")
}

// TestMigrationFromMultipleShells tests migration for both bash and zsh
func TestMigrationFromMultipleShells(t *testing.T) {
	shells := []string{"bash", "zsh"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			tmpDir := t.TempDir()
			oldHome := os.Getenv("HOME")
			defer func() { _ = os.Setenv("HOME", oldHome) }()
			require.NoError(t, os.Setenv("HOME", tmpDir))

			rcFile := filepath.Join(tmpDir, "."+shell+"rc")

			// Create legacy installation
			legacyContent := fmt.Sprintf(`# %s config
export PATH=$PATH:/usr/local/bin

%s
# Legacy hook for %s
%s

# More content
`, shell, HookMarkerStart, shell, HookMarkerEnd)

			require.NoError(t, os.WriteFile(rcFile, []byte(legacyContent), 0644))

			// Migrate
			result, err := InstallHook(shell)
			require.NoError(t, err)
			assert.True(t, result.Updated)
			assert.Contains(t, result.Message, "Migrated")

			// Verify migration succeeded
			rcContent, err := os.ReadFile(rcFile)
			require.NoError(t, err)

			assert.NotContains(t, string(rcContent), HookMarkerStart)
			assert.NotContains(t, string(rcContent), HookMarkerEnd)
			assert.Contains(t, string(rcContent), fmt.Sprintf("# %s config", shell))

			// Verify external hook file was created
			hookPath := filepath.Join(tmpDir, ".config", "dirvana", fmt.Sprintf("hook-%s.sh", shell))
			_, err = os.Stat(hookPath)
			assert.NoError(t, err, "Hook file should exist for "+shell)
		})
	}
}

// TestNoMigrationWhenNotNeeded verifies that migration doesn't happen
// when there's no legacy installation
func TestNoMigrationWhenNotNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()
	require.NoError(t, os.Setenv("HOME", tmpDir))

	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create clean RC file without legacy installation
	cleanContent := `# Clean bashrc
export PATH=$PATH:/usr/local/bin
alias ll='ls -la'
`
	require.NoError(t, os.WriteFile(rcFile, []byte(cleanContent), 0644))

	// Verify no legacy detected
	assert.False(t, HasLegacyInstall("bash"))

	// Setup should not mention migration
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated) // First install
	assert.NotContains(t, result.Message, "Migrated")
	assert.True(t, strings.Contains(result.Message, "created") || strings.Contains(result.Message, "installed"))
}

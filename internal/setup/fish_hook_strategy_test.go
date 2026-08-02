package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFishHookStrategy_Install(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Test installation
	err = strategy.Install()
	require.NoError(t, err)

	// Verify hook file was created
	assert.FileExists(t, strategy.hookPath)

	// Verify config.fish was created with proper structure
	assert.FileExists(t, strategy.rcFile)

	content, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)

	contentStr := string(content)

	// Should contain is-interactive block
	assert.Contains(t, contentStr, "if status is-interactive")
	assert.Contains(t, contentStr, "end")

	// Should contain Dirvana hook inside the block
	assert.Contains(t, contentStr, "# Dirvana")
	assert.Contains(t, contentStr, strategy.hookPath)
	assert.Contains(t, contentStr, "test -f")
	assert.Contains(t, contentStr, "and source")

	// Verify the hook is installed
	assert.True(t, strategy.IsInstalled())
}

func TestFishHookStrategy_InsertIntoExistingBlock(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Create config.fish with existing is-interactive block
	existingContent := `# Some comment
if status is-interactive
    # Commands to run in interactive sessions can go here
    starship init fish | source
end
`
	err = os.MkdirAll(filepath.Dir(strategy.rcFile), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(strategy.rcFile, []byte(existingContent), 0o644)
	require.NoError(t, err)

	// Test installation
	err = strategy.Install()
	require.NoError(t, err)

	// Read updated content
	content, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)
	contentStr := string(content)

	// Should preserve existing content
	assert.Contains(t, contentStr, "starship init fish")

	// Should contain Dirvana hook inside the existing block
	assert.Contains(t, contentStr, "# Dirvana")
	assert.Contains(t, contentStr, strategy.hookPath)

	// Verify hook is before 'end'
	hookIndex := strings.Index(contentStr, "# Dirvana")
	endIndex := strings.Index(contentStr, "end")
	assert.Less(t, hookIndex, endIndex, "Dirvana hook should be before 'end'")
}

func TestFishHookStrategy_Uninstall(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Install first
	err = strategy.Install()
	require.NoError(t, err)
	assert.True(t, strategy.IsInstalled())

	// Uninstall
	err = strategy.Uninstall()
	require.NoError(t, err)

	// Verify hook file was removed
	assert.NoFileExists(t, strategy.hookPath)

	// Verify config.fish no longer contains hook
	content, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)
	contentStr := string(content)

	assert.NotContains(t, contentStr, strategy.hookPath)
	assert.NotContains(t, contentStr, "# Dirvana")

	// Verify hook is not installed
	assert.False(t, strategy.IsInstalled())
}

func TestFishHookStrategy_NeedsUpdate(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Before installation, needs update
	assert.True(t, strategy.NeedsUpdate())

	// Install
	err = strategy.Install()
	require.NoError(t, err)

	// After installation, doesn't need update
	assert.False(t, strategy.NeedsUpdate())

	// Modify hook file
	err = os.WriteFile(strategy.hookPath, []byte("old content"), 0o644)
	require.NoError(t, err)

	// Should need update now
	assert.True(t, strategy.NeedsUpdate())
}

func TestFishHookStrategy_IdempotentInstall(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Install twice
	err = strategy.Install()
	require.NoError(t, err)

	content1, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)

	t.Logf("Content after first install:\n%s", string(content1))
	t.Logf("Hook path: %s", strategy.hookPath)

	err = strategy.Install()
	require.NoError(t, err)

	content2, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)

	t.Logf("Content after second install:\n%s", string(content2))

	// Content should be identical (no duplicate entries)
	assert.Equal(t, string(content1), string(content2))

	// Should only have one occurrence of the hook (in the source line, not counting comment)
	// Count the actual source line that loads the hook
	sourceLinePrefix := "test -f " + strategy.hookPath
	hookCount := strings.Count(string(content2), sourceLinePrefix)
	assert.Equal(t, 1, hookCount, "Should only have one occurrence of hook source line")
}

func TestFishHookStrategy_IsInstalled(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Before installation, should not be installed
	assert.False(t, strategy.IsInstalled())

	// Install
	err = strategy.Install()
	require.NoError(t, err)

	// After installation, should be installed
	assert.True(t, strategy.IsInstalled())

	// Remove hook file
	err = os.Remove(strategy.hookPath)
	require.NoError(t, err)

	// Should not be installed if hook file missing
	assert.False(t, strategy.IsInstalled())
}

func TestFishHookStrategy_GetMessage(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Before any operation, message should be default
	msg := strategy.GetMessage()
	assert.Equal(t, "✓ Dirvana hook is up to date", msg)

	// After install
	err = strategy.Install()
	require.NoError(t, err)

	msg = strategy.GetMessage()
	assert.Contains(t, msg, "Hook created")
	assert.Contains(t, msg, strategy.hookPath)
}

func TestFishHookStrategy_InsertWithoutInteractiveBlock(t *testing.T) {
	tests := []struct {
		name     string
		existing string
	}{
		{name: "content without trailing newline", existing: "set -g fish_greeting"},
		{name: "content with one trailing newline", existing: "set -g fish_greeting\n"},
		{name: "content with a blank line at the end", existing: "set -g fish_greeting\n\n"},
		{name: "empty file", existing: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHome(t)

			strategy, err := NewFishHookStrategy()
			require.NoError(t, err)

			require.NoError(t, os.MkdirAll(filepath.Dir(strategy.rcFile), 0o755))
			require.NoError(t, os.WriteFile(strategy.rcFile, []byte(tt.existing), 0o644))

			require.NoError(t, strategy.Install())

			content, err := os.ReadFile(strategy.rcFile)
			require.NoError(t, err)
			contentStr := string(content)

			// A whole is-interactive block is appended, hook included
			assert.Contains(t, contentStr, "if status is-interactive")
			assert.Contains(t, contentStr, "# Dirvana")
			assert.Contains(t, contentStr, "test -f "+strategy.hookPath)
			assert.True(t, strings.HasSuffix(contentStr, "end\n"), "block must be closed: %q", contentStr)

			// Pre-existing content is preserved and never glued to the block
			if tt.existing != "" {
				assert.Contains(t, contentStr, "set -g fish_greeting")
				assert.Contains(t, contentStr, "\n\nif status is-interactive")
			}

			assert.True(t, strategy.IsInstalled())
		})
	}
}

func TestFishHookStrategy_InsertIntoLongFormInteractiveBlock(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// The deprecated `--is-interactive` spelling must be recognized too
	existing := "if status --is-interactive\n    starship init fish | source\nend\n"
	require.NoError(t, os.MkdirAll(filepath.Dir(strategy.rcFile), 0o755))
	require.NoError(t, os.WriteFile(strategy.rcFile, []byte(existing), 0o644))

	require.NoError(t, strategy.Install())

	content, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)
	contentStr := string(content)

	// Reused rather than appending a second block
	assert.Equal(t, 1, strings.Count(contentStr, "if status"))
	assert.Contains(t, contentStr, "starship init fish")
	assert.Contains(t, contentStr, "test -f "+strategy.hookPath)
}

func TestFishHookStrategy_InstallUnreadableConfig(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// A directory where the config file is expected: reading fails with
	// something other than "not exist"
	require.NoError(t, os.MkdirAll(strategy.rcFile, 0o755))

	err = strategy.Install()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestFishHookStrategy_UninstallWithoutConfig(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Neither the hook file nor the config file exist
	require.NoError(t, strategy.Uninstall())
	assert.Equal(t, "✓ Nothing to uninstall", strategy.GetMessage())
}

func TestFishHookStrategy_UninstallUnreadableConfig(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(strategy.rcFile, 0o755))

	err = strategy.Uninstall()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestFishHookStrategy_UninstallKeepsUnrelatedLines(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)
	require.NoError(t, strategy.Install())

	// A comment that looks like ours but is followed by something else must
	// survive, along with the rest of the config
	content, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)
	updated := "# Dirvana\nset -g dirvana_like_setting 1\n" + string(content)
	require.NoError(t, os.WriteFile(strategy.rcFile, []byte(updated), 0o644))

	require.NoError(t, strategy.Uninstall())

	after, err := os.ReadFile(strategy.rcFile)
	require.NoError(t, err)
	assert.Contains(t, string(after), "set -g dirvana_like_setting 1")
	assert.NotContains(t, string(after), strategy.hookPath)
}

func TestFishHookStrategy_IsInstalledWithoutConfig(t *testing.T) {
	testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Hook file present but no config file referencing it
	require.NoError(t, os.MkdirAll(filepath.Dir(strategy.hookPath), 0o755))
	require.NoError(t, os.WriteFile(strategy.hookPath, []byte("hook"), 0o644))

	assert.False(t, strategy.IsInstalled())
}

func TestNewFishHookStrategy_NoHomeDirectory(t *testing.T) {
	t.Setenv("HOME", "")

	_, err := NewFishHookStrategy()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "home directory")
}

func TestFishHookStrategy_InstallUncreatableHookDirectory(t *testing.T) {
	home := t.TempDir()
	// A regular file where the home directory is expected: no directory of
	// the hook path can be created under it
	blocked := filepath.Join(home, "not-a-dir")
	require.NoError(t, os.WriteFile(blocked, []byte("x"), 0o644))
	t.Setenv("HOME", blocked)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	err = strategy.Install()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config directory")
}

func TestFishHookStrategy_GetRCFile(t *testing.T) {
	home := testHome(t)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	expectedRCFile := filepath.Join(home, ".config", "fish", "config.fish")
	assert.Equal(t, expectedRCFile, strategy.GetRCFile())
}

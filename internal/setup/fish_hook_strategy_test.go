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
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "dirvana-fish-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Override home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

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
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "dirvana-fish-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Override home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	strategy, err := NewFishHookStrategy()
	require.NoError(t, err)

	// Create config.fish with existing is-interactive block
	existingContent := `# Some comment
if status is-interactive
    # Commands to run in interactive sessions can go here
    starship init fish | source
end
`
	err = os.MkdirAll(filepath.Dir(strategy.rcFile), 0755)
	require.NoError(t, err)
	err = os.WriteFile(strategy.rcFile, []byte(existingContent), 0644)
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
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "dirvana-fish-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Override home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

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
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "dirvana-fish-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Override home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

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
	err = os.WriteFile(strategy.hookPath, []byte("old content"), 0644)
	require.NoError(t, err)

	// Should need update now
	assert.True(t, strategy.NeedsUpdate())
}

func TestFishHookStrategy_IdempotentInstall(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "dirvana-fish-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Override home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

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

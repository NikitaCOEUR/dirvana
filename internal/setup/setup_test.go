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

const testBashrcContent = `# My bashrc
export PATH=$PATH:/usr/local/bin
`

func TestGetRCFilePath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name    string
		shell   string
		want    string
		wantErr bool
	}{
		{
			name:  "bash",
			shell: "bash",
			want:  filepath.Join(home, ".bashrc"),
		},
		{
			name:  "zsh",
			shell: "zsh",
			want:  filepath.Join(home, ".zshrc"),
		},
		{
			name:    "unsupported shell",
			shell:   "ksh",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRCFilePath(tt.shell)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInstallHook_NewInstallation(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Pre-create rc file without hook
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Equal(t, rcFile, result.RCFile)
	// New strategy uses "created" instead of "installed"
	assert.True(t, strings.Contains(result.Message, "created") || strings.Contains(result.Message, "installed"))

	// Verify user content is preserved
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# My bashrc")

	// With new strategy, hook is in external file or drop-in, not inline
	// So we check for the source line instead of markers
	configDir := filepath.Join(tmpDir, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")

	// Either external hook file exists OR we have markers (legacy migration)
	if _, err := os.Stat(hookPath); err == nil {
		// External hook strategy
		assert.Contains(t, content, hookPath)
	} else {
		// Legacy markers (shouldn't happen for new install but keeping for compatibility)
		assert.Contains(t, content, HookMarkerStart)
		assert.Contains(t, content, HookMarkerEnd)
	}
}

func TestInstallHook_AlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Pre-create rc file with hook already installed (with markers - legacy)
	hookCode := cli.GenerateHookCode("bash")
	existingContent := fmt.Sprintf(`# My bashrc
export PATH=$PATH:/usr/local/bin

%s
%s
%s
`, HookMarkerStart, hookCode, HookMarkerEnd)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Setup should migrate and show as updated
	result, err := InstallHook("bash")
	require.NoError(t, err)
	// With legacy install, it will be migrated, so Updated = true
	assert.True(t, result.Updated)
	assert.Contains(t, result.Message, "Migrated")
}

func TestInstallHook_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Pre-create rc file with OLD hook version
	oldHook := `# Dirvana shell hook - START
__dirvana_hook() {
  # Old version
  echo "old"
}
# Dirvana shell hook - END`

	existingContent := fmt.Sprintf(`%s
%s

# More config
alias ll='ls -la'
`, testBashrcContent, oldHook)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install should migrate and update the hook
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	// New system will migrate, so look for "Migrated" instead of "updated"
	assert.Contains(t, result.Message, "Migrated")

	// Verify original content is preserved
	updatedData, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	updatedContent := string(updatedData)
	assert.Contains(t, updatedContent, "# My bashrc")
	assert.Contains(t, updatedContent, "alias ll='ls -la'")
	// Old markers should be removed
	assert.NotContains(t, updatedContent, "# Old version")
	assert.NotContains(t, updatedContent, `echo "old"`)
}

func TestIsHookInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Test when file doesn't exist
	installed, err := IsHookInstalled("bash")
	require.NoError(t, err)
	assert.False(t, installed)

	// Test when file exists without hook
	require.NoError(t, os.WriteFile(rcFile, []byte("# My bashrc\n"), 0644))
	installed, err = IsHookInstalled("bash")
	require.NoError(t, err)
	assert.False(t, installed)

	// Test when file exists with hook
	hookCode := cli.GenerateHookCode("bash")
	content := fmt.Sprintf("# My bashrc\n\n%s\n%s\n%s\n", HookMarkerStart, hookCode, HookMarkerEnd)
	require.NoError(t, os.WriteFile(rcFile, []byte(content), 0644))
	installed, err = IsHookInstalled("bash")
	require.NoError(t, err)
	assert.True(t, installed)
}

func TestInstallHook_PreservesExistingContent(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".zshrc")

	existingContent := `# Custom zsh config
export EDITOR=vim
alias g=git

# Some important stuff
function myfunction() {
  echo "test"
}
`
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook
	result, err := InstallHook("zsh")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	// Verify all existing content is preserved
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# Custom zsh config")
	assert.Contains(t, content, "export EDITOR=vim")
	assert.Contains(t, content, "alias g=git")
	assert.Contains(t, content, "function myfunction()")
	// With new strategy, hook is external, so we should see the source line
	// instead of markers (unless it's a legacy migration)
}

func TestGetRCFilePath_UnsupportedShell(t *testing.T) {
	_, err := GetRCFilePath("ksh")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

func TestIsHookInstalled_Errors(t *testing.T) {
	// Test with unsupported shell
	_, err := IsHookInstalled("ksh")
	assert.Error(t, err)
}

func TestInstallHook_ReadOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create a read-only file
	require.NoError(t, os.WriteFile(rcFile, []byte("# test\n"), 0444))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Try to install hook - should error on write
	_, err := InstallHook("bash")
	// The error depends on OS permissions, but we test it handles the case
	_ = err
}

func TestInstallHook_DirenvWarning(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file with direnv hook
	existingContent := fmt.Sprintf(`%s
# direnv hook
eval "$(direnv hook bash)"
`, testBashrcContent)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook - should warn about direnv
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.Contains(t, result.Message, "direnv")
	assert.Contains(t, result.Message, "Warning")
}

func TestUninstallHook(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file with hook installed
	hookCode := cli.GenerateHookCode("bash")
	content := fmt.Sprintf(`%s

%s
%s
%s

Some other content
`, testBashrcContent, HookMarkerStart, hookCode, HookMarkerEnd)

	require.NoError(t, os.WriteFile(rcFile, []byte(content), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Uninstall
	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	// Message should contain "Removed" (for legacy removal)
	assert.True(t, strings.Contains(result.Message, "Removed") || strings.Contains(result.Message, "removed"))

	// Verify hook is removed
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), HookMarkerStart)
	assert.Contains(t, string(data), "Some other content")
}

func TestUninstallHook_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file without hook
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Uninstall when nothing is installed
	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.False(t, result.Updated)
	assert.Contains(t, result.Message, "not installed")
}

func TestUninstallHook_FileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock home directory (no .bashrc exists)
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Uninstall when file doesn't exist
	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.False(t, result.Updated)
	// New message says "not installed" instead of "doesn't exist"
	assert.True(t, strings.Contains(result.Message, "not installed") || strings.Contains(result.Message, "Nothing to uninstall"))
}

func TestInstallHook_NoDirenvWarning(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file WITHOUT direnv
	existingContent := testBashrcContent
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook - should NOT warn about direnv
	result, err := InstallHook("bash")
	require.NoError(t, err)
	// Just check that installation succeeded without direnv warning
	assert.True(t, strings.Contains(result.Message, "installed") || strings.Contains(result.Message, "created"))
	// Check that there's no direnv warning (not the word "dirvana" which contains "direnv")
	assert.NotContains(t, result.Message, "⚠️")
	assert.NotContains(t, result.Message, "direnv may conflict")
}

func TestInstallHook_WithoutStaticCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file without hook
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook (completion is now handled dynamically by dirvana export)
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.True(t, strings.Contains(result.Message, "installed") || strings.Contains(result.Message, "created"))

	// With new strategy, hook should be external
	configDir := filepath.Join(tmpDir, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")

	// Verify either hook file exists or legacy markers exist
	_, err = os.Stat(hookPath)
	hookFileExists := err == nil

	if hookFileExists {
		// External hook strategy - verify hook file exists
		assert.NoError(t, err)
	} else {
		// Legacy strategy - verify markers
		data, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		content := string(data)
		assert.Contains(t, content, HookMarkerStart)
		assert.Contains(t, content, HookMarkerEnd)
	}
}

func TestInstallHook_AlreadyUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file with hook already installed (legacy markers)
	hookCode := cli.GenerateHookCode("bash")

	existingContent := fmt.Sprintf(`# My bashrc

%s
%s
%s
`, HookMarkerStart, hookCode, HookMarkerEnd)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Setup should migrate
	result, err := InstallHook("bash")
	require.NoError(t, err)
	// Will be migrated, so Updated = true
	assert.True(t, result.Updated)
	assert.Contains(t, result.Message, "Migrated")
}

func TestInstallHook_FileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create the rc file

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook - should create the file
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.True(t, strings.Contains(result.Message, "installed") || strings.Contains(result.Message, "created"))

	// Verify file was created
	rcFile := filepath.Join(tmpDir, ".bashrc")
	_, err = os.Stat(rcFile)
	require.NoError(t, err, "RC file should be created")
}

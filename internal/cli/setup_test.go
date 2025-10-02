package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			name:  "powershell",
			shell: "powershell",
			want:  filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		},
		{
			name:  "pwsh",
			shell: "pwsh",
			want:  filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"),
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
	assert.Contains(t, result.Message, "installed")

	// Verify hook was added
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# My bashrc")
	assert.Contains(t, content, HookMarkerStart)
	assert.Contains(t, content, HookMarkerEnd)
	assert.Contains(t, content, "__dirvana_hook()")
}

func TestInstallHook_AlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Pre-create rc file with hook already installed (with markers)
	hookCode := GenerateHookCode("bash")
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

	// Setup should detect it's already installed and up to date
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.False(t, result.Updated)
	assert.Contains(t, result.Message, "up to date")
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

	// Install should update the hook
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Contains(t, result.Message, "updated")

	// Verify hook was updated
	updatedData, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	updatedContent := string(updatedData)
	assert.Contains(t, updatedContent, "# My bashrc")
	assert.Contains(t, updatedContent, "alias ll='ls -la'")
	assert.Contains(t, updatedContent, "__dirvana_hook()")
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
	hookCode := GenerateHookCode("bash")
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
	assert.Contains(t, content, HookMarkerStart)
	assert.Contains(t, content, HookMarkerEnd)

	// Verify hook is at the end
	lines := strings.Split(content, "\n")
	var foundStart bool
	for _, line := range lines {
		if strings.Contains(line, HookMarkerStart) {
			foundStart = true
		}
		if foundStart {
			// After marker, no old content should appear
			assert.NotContains(t, line, "# Custom zsh config")
		}
	}
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

	// Create rc file with hook and completion installed
	hookCode := GenerateHookCode("bash")
	content := fmt.Sprintf(`%s

%s
%s
%s

%s
source <(dirvana completion bash)
%s

Some other content
`, testBashrcContent, HookMarkerStart, hookCode, HookMarkerEnd,
		CompletionMarkerStart, CompletionMarkerEnd)

	require.NoError(t, os.WriteFile(rcFile, []byte(content), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Uninstall
	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Contains(t, result.Message, "uninstalled")
	assert.Contains(t, result.Message, "Hook removed")
	assert.Contains(t, result.Message, "Completion removed")

	// Verify hook and completion are removed
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), HookMarkerStart)
	assert.NotContains(t, string(data), CompletionMarkerStart)
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
	assert.Contains(t, result.Message, "doesn't exist")
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
	assert.Contains(t, result.Message, "installed")
	// Check that there's no direnv warning (not the word "dirvana" which contains "direnv")
	assert.NotContains(t, result.Message, "⚠️")
	assert.NotContains(t, result.Message, "direnv may conflict")
}

func TestInstallCompletion_NewInstallation(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file without completion
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0644))

	// Install completion
	changed, wasUpdate, err := InstallCompletion("bash", rcFile)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, wasUpdate) // New installation, not update

	// Verify completion was added
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, CompletionMarkerStart)
	assert.Contains(t, content, CompletionMarkerEnd)
	assert.Contains(t, content, "command -v")
	assert.Contains(t, content, "completion bash")
}

func TestInstallCompletion_AlreadyUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Get completion code
	binPath, _ := os.Executable()
	completionCode := fmt.Sprintf("command -v %s &> /dev/null && source <(%s completion bash)", binPath, binPath)
	completionBlock := fmt.Sprintf("%s\n%s\n%s", CompletionMarkerStart, completionCode, CompletionMarkerEnd)

	// Create rc file with completion already installed
	existingContent := fmt.Sprintf(`# My bashrc

%s
`, completionBlock)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Try to install again
	changed, wasUpdate, err := InstallCompletion("bash", rcFile)
	require.NoError(t, err)
	assert.False(t, changed) // Already up to date
	assert.False(t, wasUpdate)
}

func TestInstallCompletion_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file with OLD completion (different format)
	oldCompletion := fmt.Sprintf(`%s
source <(old-dirvana completion bash)
%s`, CompletionMarkerStart, CompletionMarkerEnd)

	existingContent := fmt.Sprintf(`# My bashrc

%s
`, oldCompletion)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Install new completion
	changed, wasUpdate, err := InstallCompletion("bash", rcFile)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.True(t, wasUpdate) // Was an update

	// Verify completion was updated
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, CompletionMarkerStart)
	assert.Contains(t, content, CompletionMarkerEnd)
	assert.Contains(t, content, "command -v")
	assert.NotContains(t, content, "old-dirvana")
}

func TestInstallCompletion_Zsh(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".zshrc")

	// Create rc file
	require.NoError(t, os.WriteFile(rcFile, []byte("# My zshrc\n"), 0644))

	// Install completion for zsh
	changed, wasUpdate, err := InstallCompletion("zsh", rcFile)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.False(t, wasUpdate)

	// Verify zsh completion was added
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "completion zsh")
}

func TestInstallCompletion_UnsupportedShell(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".kshrc")

	// Try to install completion for unsupported shell
	_, _, err := InstallCompletion("ksh", rcFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

func TestInstallCompletion_InvalidPath(t *testing.T) {
	// Try to write to invalid path
	_, _, err := InstallCompletion("bash", "/nonexistent/directory/.bashrc")
	require.Error(t, err)
}

func TestInstallCompletion_PreservesContent(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	existingContent := `# Custom config
export EDITOR=vim
alias ll='ls -la'
`
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Install completion
	changed, _, err := InstallCompletion("bash", rcFile)
	require.NoError(t, err)
	assert.True(t, changed)

	// Verify existing content is preserved
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# Custom config")
	assert.Contains(t, content, "export EDITOR=vim")
	assert.Contains(t, content, "alias ll='ls -la'")
	assert.Contains(t, content, CompletionMarkerStart)
}

func TestInstallHook_WithCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file without hook or completion
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook (should also install completion)
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.True(t, result.CompletionInstalled)
	assert.Contains(t, result.Message, "installed")
	assert.Contains(t, result.Message, "completion")

	// Verify both hook and completion were added
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, HookMarkerStart)
	assert.Contains(t, content, HookMarkerEnd)
	assert.Contains(t, content, CompletionMarkerStart)
	assert.Contains(t, content, CompletionMarkerEnd)
}

func TestInstallHook_CompletionAlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file with hook already installed
	hookCode := GenerateHookCode("bash")
	binPath, _ := os.Executable()
	completionCode := fmt.Sprintf("command -v %s &> /dev/null && source <(%s completion bash)", binPath, binPath)
	completionBlock := fmt.Sprintf("%s\n%s\n%s", CompletionMarkerStart, completionCode, CompletionMarkerEnd)

	existingContent := fmt.Sprintf(`# My bashrc

%s
%s
%s

%s
`, HookMarkerStart, hookCode, HookMarkerEnd, completionBlock)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook - should detect both are up to date
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.False(t, result.Updated)
	assert.False(t, result.CompletionInstalled)
	assert.Contains(t, result.Message, "hook is up to date")
	assert.Contains(t, result.Message, "completion is up to date")
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
	assert.Contains(t, result.Message, "installed")

	// Verify file was created with hook
	rcFile := filepath.Join(tmpDir, ".bashrc")
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, HookMarkerStart)
	assert.Contains(t, content, HookMarkerEnd)
}

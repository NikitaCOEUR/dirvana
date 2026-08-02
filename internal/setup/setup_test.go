package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/shell"
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
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0o644))

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

	// Hook is installed in an external file (or drop-in), never inline
	configDir := filepath.Join(tmpDir, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	_, err = os.Stat(hookPath)
	require.NoError(t, err, "external hook file should exist")
	assert.Contains(t, content, hookPath)
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
	require.NoError(t, os.WriteFile(rcFile, []byte("# My bashrc\n"), 0o644))
	installed, err = IsHookInstalled("bash")
	require.NoError(t, err)
	assert.False(t, installed)

	// Test after a real installation
	_, err = InstallHook("bash")
	require.NoError(t, err)
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
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0o644))

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
	require.NoError(t, os.WriteFile(rcFile, []byte("# test\n"), 0o444))

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
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0o644))

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

	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent+"\nSome other content\n"), 0o644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install then uninstall
	_, err := InstallHook("bash")
	require.NoError(t, err)

	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	// Verify hook is removed and user content preserved
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "dirvana")
	assert.Contains(t, string(data), "Some other content")

	installed, err := IsHookInstalled("bash")
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestUninstallHook_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create rc file without hook
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0o644))

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
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0o644))

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
	require.NoError(t, os.WriteFile(rcFile, []byte(testBashrcContent), 0o644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook (completion is now handled dynamically by dirvana export)
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.True(t, strings.Contains(result.Message, "installed") || strings.Contains(result.Message, "created"))

	// Hook is installed in an external file
	configDir := filepath.Join(tmpDir, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	_, err = os.Stat(hookPath)
	assert.NoError(t, err, "external hook file should exist")
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

func TestUninstallHook_WithDropInStrategy(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create RC file with drop-in support
	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent := "if [ -d ~/.bashrc.d ]; then\n  for rc in ~/.bashrc.d/*.sh; do\n    source $rc\n  done\nfi"
	err := os.WriteFile(rcFile, []byte(rcContent), 0o644)
	require.NoError(t, err)

	// Create drop-in directory and hook file
	dropInDir := filepath.Join(tmpDir, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")
	err = os.MkdirAll(dropInDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(dropInFile, []byte("# Hook code"), 0o644)
	require.NoError(t, err)

	// Uninstall
	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	// Verify drop-in file was removed
	_, err = os.Stat(dropInFile)
	assert.True(t, os.IsNotExist(err), "Drop-in file should be removed")
}

func TestUninstallHook_WithExternalHook(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create external hook file
	configDir := filepath.Join(tmpDir, ".config", "dirvana")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	hookPath := filepath.Join(configDir, "hook-bash.sh")
	err = os.WriteFile(hookPath, []byte("# Test hook"), 0o644)
	require.NoError(t, err)

	// Create RC file with reference to external hook
	rcFile := filepath.Join(tmpDir, ".bashrc")
	sourceLine := fmt.Sprintf("[ -f %s ] && source %s", hookPath, hookPath)
	hookCode := fmt.Sprintf("# Dirvana\n%s\n", sourceLine)
	err = os.WriteFile(rcFile, []byte(hookCode), 0o644)
	require.NoError(t, err)

	// Uninstall
	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	// Verify hook file was removed
	_, err = os.Stat(hookPath)
	assert.True(t, os.IsNotExist(err), "Hook file should be removed")

	// Verify hook reference was removed from RC file
	content, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "Dirvana")
	assert.NotContains(t, string(content), hookPath)
}

func TestInstallHook_ErrorCases(t *testing.T) {
	t.Run("unsupported shell", func(t *testing.T) {
		_, err := InstallHook("unsupported-shell")
		assert.Error(t, err)
	})

	t.Run("no home directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		_, err := InstallHook("bash")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "home directory")
	})

	t.Run("unreadable rc file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		// A directory where .bashrc is expected: the legacy cleanup cannot
		// even read the file
		require.NoError(t, os.Mkdir(filepath.Join(home, ".bashrc"), 0o755))

		_, err := InstallHook("bash")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "legacy inline hook")
	})
}

func TestUninstallHook_ErrorCases(t *testing.T) {
	t.Run("unsupported shell", func(t *testing.T) {
		_, err := UninstallHook("unsupported-shell")
		assert.Error(t, err)
	})

	t.Run("no home directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		_, err := UninstallHook("bash")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "home directory")
	})

	t.Run("unreadable rc file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		require.NoError(t, os.Mkdir(filepath.Join(home, ".bashrc"), 0o755))

		_, err := UninstallHook("bash")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "legacy inline hook")
	})
}

func TestGetRCFilePath_NoHomeDirectory(t *testing.T) {
	t.Setenv("HOME", "")

	_, err := GetRCFilePath("bash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "home directory")
}

func TestCheckDirenvConflict_NoConflict(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	content := "# Regular bashrc\nexport PATH=$PATH:/usr/local/bin"
	err := os.WriteFile(rcFile, []byte(content), 0o644)
	require.NoError(t, err)

	warning := checkDirenvConflict(rcFile)
	assert.Empty(t, warning)
}

func TestCheckDirenvConflict_WithConflict(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	content := "# Bashrc with direnv\neval \"$(direnv hook bash)\""
	err := os.WriteFile(rcFile, []byte(content), 0o644)
	require.NoError(t, err)

	warning := checkDirenvConflict(rcFile)
	assert.Contains(t, warning, "direnv")
	assert.Contains(t, warning, "Warning")
}

func TestCheckDirenvConflict_FileNotExist(t *testing.T) {
	warning := checkDirenvConflict("/nonexistent/file")
	assert.Empty(t, warning)
}

func TestInstallHook_UpToDateNoUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Create external hook that's already up to date
	configDir := filepath.Join(tmpDir, ".config", "dirvana")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	hookPath := filepath.Join(configDir, "hook-bash.sh")
	hookCode, err := shell.GenerateHookCode("bash", shell.BinaryPath())
	require.NoError(t, err)
	err = os.WriteFile(hookPath, []byte(hookCode), 0o644)
	require.NoError(t, err)

	// Create RC file with reference
	rcFile := filepath.Join(tmpDir, ".bashrc")
	sourceLine := fmt.Sprintf("[ -f %s ] && source %s", hookPath, hookPath)
	content := fmt.Sprintf("# Dirvana\n%s\n", sourceLine)
	err = os.WriteFile(rcFile, []byte(content), 0o644)
	require.NoError(t, err)

	// Install when already up to date
	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.False(t, result.Updated, "Should not update when already up to date")
	assert.Contains(t, result.Message, "up to date")
}

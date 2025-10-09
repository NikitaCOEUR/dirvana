package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestZshHookContent verifies that zsh-specific hook code is correctly generated
func TestZshHookContent(t *testing.T) {
	hookCode := cli.GenerateHookCode("zsh")

	// Must have zsh-specific features
	assert.Contains(t, hookCode, "__dirvana_hook()", "Must define __dirvana_hook function")
	assert.Contains(t, hookCode, "autoload -U add-zsh-hook", "Must autoload add-zsh-hook")
	assert.Contains(t, hookCode, "add-zsh-hook chpwd __dirvana_hook", "Must register chpwd hook")

	// Must have safety checks (same as bash)
	assert.Contains(t, hookCode, "DIRVANA_ENABLED", "Must check DIRVANA_ENABLED")
	assert.Contains(t, hookCode, "command -v", "Must check if dirvana command exists")
	assert.Contains(t, hookCode, "[[ ! -t 0 ]]", "Must check if stdin is terminal")

	// Must NOT have bash-specific code
	assert.NotContains(t, hookCode, "PROMPT_COMMAND", "Should not contain bash PROMPT_COMMAND")
}

// TestBashHookContent verifies that bash-specific hook code is correctly generated
func TestBashHookContent(t *testing.T) {
	hookCode := cli.GenerateHookCode("bash")

	// Must have bash-specific features
	assert.Contains(t, hookCode, "__dirvana_hook()", "Must define __dirvana_hook function")
	assert.Contains(t, hookCode, "PROMPT_COMMAND", "Must set PROMPT_COMMAND")

	// Must have safety checks
	assert.Contains(t, hookCode, "DIRVANA_ENABLED", "Must check DIRVANA_ENABLED")
	assert.Contains(t, hookCode, "command -v", "Must check if dirvana command exists")
	assert.Contains(t, hookCode, "[[ ! -t 0 ]]", "Must check if stdin is terminal")

	// Must NOT have zsh-specific code
	assert.NotContains(t, hookCode, "add-zsh-hook", "Should not contain zsh add-zsh-hook")
	assert.NotContains(t, hookCode, "autoload", "Should not contain zsh autoload")
}

// TestBothShellsUseSameStrategy verifies that both bash and zsh can be installed
// simultaneously using the same strategy pattern
func TestBothShellsUseSameStrategy(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	require.NoError(t, os.Setenv("HOME", tmpDir))

	// Create RC files for both shells
	bashRC := filepath.Join(tmpDir, ".bashrc")
	zshRC := filepath.Join(tmpDir, ".zshrc")
	require.NoError(t, os.WriteFile(bashRC, []byte("# bash\n"), 0644))
	require.NoError(t, os.WriteFile(zshRC, []byte("# zsh\n"), 0644))

	// Install for bash
	bashResult, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, bashResult.Updated)

	// Install for zsh
	zshResult, err := InstallHook("zsh")
	require.NoError(t, err)
	assert.True(t, zshResult.Updated)

	// Both should use external hook strategy (no drop-in support in this test)
	bashHookPath := filepath.Join(tmpDir, ".config", "dirvana", "hook-bash.sh")
	zshHookPath := filepath.Join(tmpDir, ".config", "dirvana", "hook-zsh.sh")

	_, err = os.Stat(bashHookPath)
	assert.NoError(t, err, "Bash hook file should exist")

	_, err = os.Stat(zshHookPath)
	assert.NoError(t, err, "Zsh hook file should exist")

	// Both RC files should have their respective source lines
	bashContent, _ := os.ReadFile(bashRC)
	zshContent, _ := os.ReadFile(zshRC)

	assert.Contains(t, string(bashContent), bashHookPath)
	assert.Contains(t, string(zshContent), zshHookPath)

	// Verify hook files have shell-specific content
	bashHookContent, _ := os.ReadFile(bashHookPath)
	zshHookContent, _ := os.ReadFile(zshHookPath)

	assert.Contains(t, string(bashHookContent), "PROMPT_COMMAND", "Bash hook should have PROMPT_COMMAND")
	assert.Contains(t, string(zshHookContent), "add-zsh-hook", "Zsh hook should have add-zsh-hook")
}

// TestGetRCFilePath_Zsh verifies zsh RC file path is correct
func TestGetRCFilePath_Zsh(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	rcFile, err := GetRCFilePath("zsh")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".zshrc"), rcFile)
}

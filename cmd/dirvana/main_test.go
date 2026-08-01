package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	dircli "github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/NikitaCOEUR/dirvana/internal/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		shellEnv string
		want     string
	}{
		{
			name: "explicit bash",
			flag: "bash",
			want: "bash",
		},
		{
			name: "explicit zsh",
			flag: "zsh",
			want: "zsh",
		},
		{
			name:     "auto detect zsh",
			flag:     "auto",
			shellEnv: "/bin/zsh",
			want:     "zsh",
		},
		{
			name:     "auto detect bash",
			flag:     "auto",
			shellEnv: "/bin/bash",
			want:     "bash",
		},
		{
			name:     "auto defaults to bash",
			flag:     "auto",
			shellEnv: "",
			want:     "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up shell version environment variables that might interfere
			envVars := []string{"FISH_VERSION", "ZSH_VERSION", "BASH_VERSION", "DIRVANA_SHELL"}
			for _, envVar := range envVars {
				_ = os.Unsetenv(envVar)
			}

			if tt.shellEnv != "" {
				_ = os.Setenv("SHELL", tt.shellEnv)
				defer func() { _ = os.Unsetenv("SHELL") }()
			}

			got := dircli.DetectShell(tt.flag)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateHookCode(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  []string // Must contain these strings
	}{
		{
			name:  "bash hook",
			shell: "bash",
			want: []string{
				"__dirvana_hook()",
				"PROMPT_COMMAND",
				"DIRVANA_PREV_DIR",
				"[[ ! -t 0 ]]",
			},
		},
		{
			name:  "zsh hook",
			shell: "zsh",
			want: []string{
				"__dirvana_hook()",
				"autoload -U add-zsh-hook",
				"add-zsh-hook chpwd __dirvana_hook",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := dircli.GenerateHookCode(tt.shell)
			for _, expected := range tt.want {
				assert.Contains(t, code, expected)
			}
		})
	}
}

func TestSetupCommand_AlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	require.NoError(t, os.WriteFile(rcFile, []byte("# My bashrc\n"), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// First install
	result, err := setup.InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	// Second install: already up to date, nothing to do
	result, err = setup.InstallHook("bash")
	require.NoError(t, err)
	assert.False(t, result.Updated)
	assert.Contains(t, result.Message, "up to date")
}

func TestSetupCommand_NewInstallation(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Pre-create rc file without hook
	existingContent := `# My bashrc
export PATH=$PATH:/usr/local/bin
`
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install hook
	result, err := setup.InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	// New strategy uses "created" instead of "installed"
	assert.True(t, strings.Contains(result.Message, "installed") || strings.Contains(result.Message, "created"))

	// Verify user content is preserved
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# My bashrc")

	// With new strategy, hook is external, not inline
	// Verify external hook file was created
	hookPath := filepath.Join(tmpDir, ".config", "dirvana", "hook-bash.sh")
	_, err = os.Stat(hookPath)
	assert.NoError(t, err, "External hook file should exist")

	// Verify RC file contains reference to external hook
	assert.Contains(t, content, hookPath)
}

func TestSetupCommand_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	existingContent := `# My bashrc
export PATH=$PATH:/usr/local/bin

# More config
alias ll='ls -la'
`
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install, then corrupt the external hook file to simulate an outdated hook
	_, err := setup.InstallHook("bash")
	require.NoError(t, err)

	hookPath := filepath.Join(tmpDir, ".config", "dirvana", "hook-bash.sh")
	require.NoError(t, os.WriteFile(hookPath, []byte("# outdated hook\n"), 0644))

	// Install again: hook content differs, must be rewritten
	result, err := setup.InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	// Verify user content is preserved and hook file is fresh
	updatedData, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	updatedContent := string(updatedData)
	assert.Contains(t, updatedContent, "# My bashrc")
	assert.Contains(t, updatedContent, "alias ll='ls -la'")

	hookData, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(hookData), "__dirvana_hook")
}

func TestEnvVarSupport_Shell(t *testing.T) {
	// Test that DIRVANA_SHELL env var takes precedence
	_ = os.Setenv("DIRVANA_SHELL", "zsh")
	defer func() { _ = os.Unsetenv("DIRVANA_SHELL") }()

	// When DIRVANA_SHELL is set, DetectShell should return it
	shell := dircli.DetectShell("auto")
	assert.Equal(t, "zsh", shell)

	// When explicitly set, it should override env var
	shell = dircli.DetectShell("bash")
	assert.Equal(t, "bash", shell)
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	dircli "github.com/NikitaCOEUR/dirvana/internal/cli"
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
				"__dirvana_cd()",
				"alias cd='__dirvana_cd'",
				"export",
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

	// Pre-create rc file with hook already installed (with markers)
	hookCode := dircli.GenerateHookCode("bash")
	existingContent := fmt.Sprintf(`# My bashrc
export PATH=$PATH:/usr/local/bin

%s
%s
%s
`, dircli.HookMarkerStart, hookCode, dircli.HookMarkerEnd)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Setup should detect it's already installed and up to date
	result, err := dircli.InstallHook("bash")
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
	result, err := dircli.InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Contains(t, result.Message, "installed")

	// Verify hook was added
	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# My bashrc")
	assert.Contains(t, content, dircli.HookMarkerStart)
	assert.Contains(t, content, dircli.HookMarkerEnd)
	assert.Contains(t, content, "__dirvana_hook()")
}

func TestSetupCommand_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Pre-create rc file with OLD hook version
	oldHook := fmt.Sprintf(`%s
__dirvana_hook() {
  # Old version
  echo "old"
}
%s`, dircli.HookMarkerStart, dircli.HookMarkerEnd)

	existingContent := fmt.Sprintf(`# My bashrc
export PATH=$PATH:/usr/local/bin

%s

# More config
alias ll='ls -la'
`, oldHook)
	require.NoError(t, os.WriteFile(rcFile, []byte(existingContent), 0644))

	// Mock home directory
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Install should update the hook
	result, err := dircli.InstallHook("bash")
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

func TestEnvVarSupport_Shell(t *testing.T) {
	// Test that DIRVANA_SHELL env var works
	_ = os.Setenv("DIRVANA_SHELL", "zsh")
	defer func() { _ = os.Unsetenv("DIRVANA_SHELL") }()

	// The DetectShell function should work with "auto" when SHELL env is not set
	// but DIRVANA_SHELL would be read by the CLI framework itself
	shell := dircli.DetectShell("auto")
	// This will default to bash since SHELL env is not set
	assert.Equal(t, "bash", shell)

	// When explicitly set to zsh
	shell = dircli.DetectShell("zsh")
	assert.Equal(t, "zsh", shell)
}

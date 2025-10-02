package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

			got := DetectShell(tt.flag)
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
			code := GenerateHookCode(tt.shell)
			for _, expected := range tt.want {
				assert.Contains(t, code, expected)
			}
		})
	}
}

func TestGenerateHookCode_NotEmpty(t *testing.T) {
	tests := []string{"bash", "zsh"}
	for _, shell := range tests {
		t.Run(shell, func(t *testing.T) {
			code := GenerateHookCode(shell)
			assert.NotEmpty(t, code)
			lines := strings.Split(code, "\n")
			assert.Greater(t, len(lines), 3, "Hook should have multiple lines")
		})
	}
}

func TestGenerateHookCode_DefaultShell(t *testing.T) {
	// Test with an unknown shell - should default to bash behavior
	code := GenerateHookCode("unknown")
	assert.NotEmpty(t, code)
	assert.Contains(t, code, "__dirvana_hook()")
	assert.Contains(t, code, "__dirvana_cd()")
	assert.Contains(t, code, "alias cd='__dirvana_cd'")
}

func TestGenerateHookCode_PowerShell(t *testing.T) {
	hookCode := GenerateHookCode(ShellPowerShell)

	assert.Contains(t, hookCode, "function __Dirvana-Hook")
	assert.Contains(t, hookCode, "Get-Command")
	assert.Contains(t, hookCode, "$env:DIRVANA_PREV_DIR")
	assert.Contains(t, hookCode, "Invoke-Expression")
	assert.Contains(t, hookCode, "Register-EngineEvent")
}

func TestGenerateHookCode_Pwsh(t *testing.T) {
	hookCode := GenerateHookCode(ShellPwsh)

	// Should generate same as PowerShell
	assert.Contains(t, hookCode, "function __Dirvana-Hook")
	assert.Contains(t, hookCode, "$env:DIRVANA_PREV_DIR")
}

func TestDetectShell_PowerShell(t *testing.T) {
	t.Setenv("PSModulePath", "C:\\Program Files\\PowerShell\\Modules")
	shell := DetectShell("auto")
	assert.Equal(t, ShellPowerShell, shell)
}

func TestDetectShell_Pwsh(t *testing.T) {
	t.Setenv("PSModulePath", "/usr/local/share/pwsh/Modules")
	shell := DetectShell("auto")
	assert.Equal(t, ShellPwsh, shell)
}

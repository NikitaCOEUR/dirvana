package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	dirvanaBinaryName = "dirvana"
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
				"PROMPT_COMMAND",
				"DIRVANA_PREV_DIR",
				"dirvana export",
				"eval",
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
	assert.Contains(t, code, "PROMPT_COMMAND")
	assert.Contains(t, code, "dirvana export")
	assert.Contains(t, code, "eval")
}

func TestDetectShellFromParentProcess(t *testing.T) {
	// This function reads /proc/$PPID/cmdline on Linux
	// On other systems or if the file doesn't exist, it returns ""
	result := detectShellFromParentProcess()

	// Should return either a valid shell name or empty string
	assert.Contains(t, []string{"", ShellBash, ShellZsh, ShellFish}, result)
}

func TestDetectShell_WithDirvanaShellEnv(t *testing.T) {
	t.Setenv("DIRVANA_SHELL", "zsh")
	t.Setenv("SHELL", "/bin/bash") // Should be ignored

	shell := DetectShell("auto")
	assert.Equal(t, "zsh", shell, "DIRVANA_SHELL should take priority")
}

func TestDetectShell_FallbackOrder(t *testing.T) {
	// Clear all env vars
	_ = os.Unsetenv("DIRVANA_SHELL")
	_ = os.Unsetenv("SHELL")
	_ = os.Unsetenv("PSModulePath")
	_ = os.Unsetenv("FISH_VERSION")
	_ = os.Unsetenv("ZSH_VERSION")
	_ = os.Unsetenv("BASH_VERSION")

	// With no environment variables, should default to bash
	shell := DetectShell("auto")
	assert.Equal(t, ShellBash, shell, "Should default to bash when no detection works")
}

func TestDetectShell_VersionVariables(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		envVal  string
		want    string
	}{
		{
			name:   "detect fish via FISH_VERSION",
			envVar: "FISH_VERSION",
			envVal: "3.6.0",
			want:   ShellFish,
		},
		{
			name:   "detect zsh via ZSH_VERSION",
			envVar: "ZSH_VERSION",
			envVal: "5.9",
			want:   ShellZsh,
		},
		{
			name:   "detect bash via BASH_VERSION",
			envVar: "BASH_VERSION",
			envVal: "5.1.16",
			want:   ShellBash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all detection env vars
			_ = os.Unsetenv("DIRVANA_SHELL")
			_ = os.Unsetenv("FISH_VERSION")
			_ = os.Unsetenv("ZSH_VERSION")
			_ = os.Unsetenv("BASH_VERSION")
			_ = os.Unsetenv("SHELL")

			// Set the specific version variable
			t.Setenv(tt.envVar, tt.envVal)

			shell := DetectShell("auto")
			assert.Equal(t, tt.want, shell)
		})
	}
}

func TestDetectShell_PriorityOrder(t *testing.T) {
	// DIRVANA_SHELL should take priority over version variables
	t.Run("DIRVANA_SHELL overrides version variables", func(t *testing.T) {
		_ = os.Unsetenv("DIRVANA_SHELL")
		_ = os.Unsetenv("FISH_VERSION")
		_ = os.Unsetenv("ZSH_VERSION")
		_ = os.Unsetenv("BASH_VERSION")

		t.Setenv("DIRVANA_SHELL", "bash")
		t.Setenv("FISH_VERSION", "3.6.0")
		t.Setenv("SHELL", "/bin/zsh")

		shell := DetectShell("auto")
		assert.Equal(t, ShellBash, shell, "DIRVANA_SHELL should take priority")
	})

	// Version variables should take priority over SHELL env var
	t.Run("version variables override SHELL env var", func(t *testing.T) {
		_ = os.Unsetenv("DIRVANA_SHELL")
		_ = os.Unsetenv("FISH_VERSION")
		_ = os.Unsetenv("ZSH_VERSION")
		_ = os.Unsetenv("BASH_VERSION")

		t.Setenv("ZSH_VERSION", "5.9")
		t.Setenv("SHELL", "/bin/bash")

		shell := DetectShell("auto")
		assert.Equal(t, ShellZsh, shell, "ZSH_VERSION should take priority over SHELL")
	})
}

func TestParseShellFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "bash absolute path",
			path: "/bin/bash",
			want: ShellBash,
		},
		{
			name: "zsh absolute path",
			path: "/usr/bin/zsh",
			want: ShellZsh,
		},
		{
			name: "fish absolute path",
			path: "/usr/local/bin/fish",
			want: ShellFish,
		},
		{
			name: "bash in homebrew",
			path: "/opt/homebrew/bin/bash",
			want: ShellBash,
		},
		{
			name: "zsh with oh-my-zsh path",
			path: "/home/user/.oh-my-zsh/bin/zsh",
			want: ShellZsh,
		},
		{
			name: "uppercase path",
			path: "/BIN/BASH",
			want: ShellBash,
		},
		{
			name: "mixed case",
			path: "/usr/bin/ZsH",
			want: ShellZsh,
		},
		{
			name: "unknown shell",
			path: "/bin/sh",
			want: "",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseShellFromPath(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseShellFromCmdline(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		want    string
	}{
		{
			name:    "zsh in path",
			cmdline: "/usr/bin/zsh",
			want:    ShellZsh,
		},
		{
			name:    "bash in path",
			cmdline: "/bin/bash",
			want:    ShellBash,
		},
		{
			name:    "zsh with arguments",
			cmdline: "/usr/local/bin/zsh\x00-l",
			want:    ShellZsh,
		},
		{
			name:    "bash with arguments",
			cmdline: "/bin/bash\x00--login",
			want:    ShellBash,
		},
		{
			name:    "unknown shell",
			cmdline: "/bin/sh",
			want:    "",
		},
		{
			name:    "fish shell",
			cmdline: "/usr/bin/fish",
			want:    ShellFish,
		},
		{
			name:    "empty cmdline",
			cmdline: "",
			want:    "",
		},
		{
			name:    "zsh in middle of path",
			cmdline: "/home/user/.oh-my-zsh/bin/zsh",
			want:    ShellZsh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseShellFromCmdline(tt.cmdline)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetBinaryPath(t *testing.T) {
	// Should return a non-empty path
	path := getBinaryPath()
	assert.NotEmpty(t, path)

	// Should either be an absolute path or "dirvana" (fallback)
	if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "\\") && path != dirvanaBinaryName {
		t.Errorf("Unexpected binary path format: %s", path)
	}
}

func TestGetBinaryPath_Fallback(t *testing.T) {
	// We can't easily make os.Executable() fail, but we can verify
	// that the function handles both success and fallback paths
	path := getBinaryPath()

	// The result should be either:
	// 1. A valid executable path (contains "/" or "\\")
	// 2. The fallback value "dirvana"
	assert.NotEmpty(t, path, "getBinaryPath should never return empty string")

	// Verify the path is usable in a hook command
	assert.True(t,
		strings.Contains(path, "/") ||
			strings.Contains(path, "\\") ||
			path == dirvanaBinaryName,
		"Path should be absolute or fallback to 'dirvana'")
}

func TestDetectShell_ParentProcessDetection(t *testing.T) {
	// This test verifies that detectShellFromParentProcess is called
	// Clear DIRVANA_SHELL, SHELL, and version variables to force parent process detection
	_ = os.Unsetenv("DIRVANA_SHELL")
	_ = os.Unsetenv("SHELL")
	_ = os.Unsetenv("FISH_VERSION")
	_ = os.Unsetenv("ZSH_VERSION")
	_ = os.Unsetenv("BASH_VERSION")

	// Call DetectShell with auto
	shell := DetectShell("auto")

	// On Linux, if running under bash/zsh/fish, detectShellFromParentProcess
	// might succeed. Otherwise it falls back to bash.
	// The test passes if we get a valid shell type
	assert.Contains(t, []string{ShellBash, ShellZsh, ShellFish}, shell,
		"Should return bash, zsh, or fish (either from parent detection or fallback)")
}

func TestGenerateHookCode_ContainsBinaryPath(t *testing.T) {
	binPath := getBinaryPath()

	tests := []string{ShellBash, ShellZsh}
	for _, shell := range tests {
		t.Run(shell, func(t *testing.T) {
			code := GenerateHookCode(shell)

			// The hook code should reference the binary path
			// Either the full path or "dirvana" fallback
			containsPath := strings.Contains(code, binPath) || strings.Contains(code, dirvanaBinaryName)
			assert.True(t, containsPath, "Hook code should contain binary path reference")
		})
	}
}

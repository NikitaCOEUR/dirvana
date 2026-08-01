// Package cli provides CLI-related functionality for Dirvana.
package cli

import (
	"fmt"
	"os"
	"strings"

	shellpkg "github.com/NikitaCOEUR/dirvana/internal/shell"
)

// Supported shells
const (
	// ShellBash represents bash shell
	ShellBash = "bash"
	// ShellZsh represents zsh shell
	ShellZsh = "zsh"
	// ShellFish represents fish shell
	ShellFish = "fish"
)

// DetectShell determines the shell type based on the flag or environment.
// Detection priority:
// 1. Explicit shell flag (if not "auto")
// 2. DIRVANA_SHELL env var (set by hook, most reliable)
// 3. Shell-specific version variables (FISH_VERSION, ZSH_VERSION, BASH_VERSION)
// 4. Parent process detection (Linux/macOS via /proc)
// 5. SHELL env var (login shell, less reliable)
// 6. Default to bash
func DetectShell(shellFlag string) string {
	if shellFlag != "auto" {
		return shellFlag
	}

	// Try DIRVANA_SHELL env var (set by hook)
	if dirvanaShell := os.Getenv("DIRVANA_SHELL"); dirvanaShell != "" {
		return dirvanaShell
	}

	// Try shell-specific version variables (most reliable runtime detection)
	if os.Getenv("FISH_VERSION") != "" {
		return ShellFish
	}
	if os.Getenv("ZSH_VERSION") != "" {
		return ShellZsh
	}
	if os.Getenv("BASH_VERSION") != "" {
		return ShellBash
	}

	// Try to detect from parent process (works on Linux/macOS)
	if parentShell := detectShellFromParentProcess(); parentShell != "" {
		return parentShell
	}

	// Try SHELL env var (usually set to login shell, less reliable)
	if shell := os.Getenv("SHELL"); shell != "" {
		return parseShellName(shell)
	}

	// Default to bash
	return ShellBash
}

// parseShellName detects the shell type from any string naming it: a path
// like "/usr/bin/fish" or a process command line
func parseShellName(s string) string {
	s = strings.ToLower(s)
	if strings.Contains(s, "fish") {
		return ShellFish
	}
	if strings.Contains(s, "zsh") {
		return ShellZsh
	}
	if strings.Contains(s, "bash") {
		return ShellBash
	}
	return ""
}

// detectShellFromParentProcess tries to detect the shell by reading the parent process name
func detectShellFromParentProcess() string {
	// Try to read /proc/$PPID/cmdline (Linux)
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getppid()))
	if err == nil {
		return parseShellName(string(cmdline))
	}

	// On macOS, we could use ps, but that's more complex
	// For now, return empty to fall back to other detection methods
	return ""
}

// getBinaryPath returns the path to the running dirvana binary so that
// generated hooks work even when dirvana is not on $PATH. Falls back to
// the bare name if the executable path cannot be resolved.
func getBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "dirvana"
	}
	return exe
}

// GenerateHookCode generates the shell hook code for the specified shell
// using the embedded templates from internal/shell.
func GenerateHookCode(shell string) (string, error) {
	return shellpkg.GenerateHookCode(shell, getBinaryPath())
}

// Package cli provides CLI-related functionality for Dirvana.
package cli

import (
	"fmt"
	"os"
	"strings"

	shellpkg "github.com/NikitaCOEUR/dirvana/internal/shell"
)

const (
	// ShellBash represents bash shell
	ShellBash = "bash"
	// ShellZsh represents zsh shell
	ShellZsh = "zsh"
)

// DetectShell determines the shell type based on the flag or environment.
func DetectShell(shellFlag string) string {
	if shellFlag != "auto" {
		return shellFlag
	}

	// First, try DIRVANA_SHELL env var (set by hook)
	if dirvanaShell := os.Getenv("DIRVANA_SHELL"); dirvanaShell != "" {
		return dirvanaShell
	}

	// Second, try to detect from parent process (works on Linux/macOS)
	if parentShell := detectShellFromParentProcess(); parentShell != "" {
		return parentShell
	}

	// Third, try SHELL env var (usually set to login shell)
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return ShellZsh
	}
	if strings.Contains(shell, "bash") {
		return ShellBash
	}

	// Default to bash
	return ShellBash
}

// parseShellFromCmdline parses a command line string to detect the shell type
// This is a pure function that can be easily tested
func parseShellFromCmdline(cmdline string) string {
	if strings.Contains(cmdline, "zsh") {
		return ShellZsh
	}
	if strings.Contains(cmdline, "bash") {
		return ShellBash
	}
	return ""
}

// detectShellFromParentProcess tries to detect the shell by reading the parent process name
func detectShellFromParentProcess() string {
	// This works on Linux and macOS
	ppid := os.Getppid()

	// Try to read /proc/$PPID/cmdline (Linux)
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", ppid))
	if err == nil {
		return parseShellFromCmdline(string(cmdline))
	}

	// On macOS, we could use ps, but that's more complex
	// For now, return empty to fall back to other detection methods
	return ""
}

// getBinaryPath returns the path to the dirvana binary, with fallback
func getBinaryPath() string {
	return "dirvana" // Fallback to just "dirvana", assuming it's in PATH
}

// GenerateHookCode generates the shell hook code for the specified shell.
// This is now a thin wrapper around shell.GenerateHookCode which uses embedded templates.
func GenerateHookCode(shell string) string {
	binPath := getBinaryPath()

	// Use the template-based generator from internal/shell
	code, err := shellpkg.GenerateHookCode(shell, binPath)
	if err != nil {
		// Fallback to bash if there's an error
		code, _ = shellpkg.GenerateHookCode("bash", binPath)
	}

	return code
}

// Package cli provides CLI-related functionality for Dirvana.
package cli

import (
	"fmt"
	"os"
	"strings"
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
	binPath, err := os.Executable()
	if err != nil {
		return "dirvana" // Fallback to PATH
	}
	return binPath
}

// GenerateHookCode generates the shell hook code for the specified shell.
func GenerateHookCode(shell string) string {
	binPath := getBinaryPath()

	switch shell {
	case ShellZsh:
		return fmt.Sprintf(`__dirvana_hook() {
  # Check if Dirvana is disabled
  if [[ "${DIRVANA_ENABLED:-true}" == "false" ]]; then
    return 0
  fi

  # Check if dirvana command exists
  if ! command -v %s &> /dev/null; then
    return 0
  fi

  # Don't run if stdin is not the terminal (prevents TUI interference)
  if [[ ! -t 0 ]]; then
    return 0
  fi

  local shell_code
  shell_code=$(%s export --prev "${DIRVANA_PREV_DIR:-}")
  local exit_code=$?
  export DIRVANA_PREV_DIR="$PWD"
  [[ $exit_code -eq 0 && -n "$shell_code" ]] && eval "$shell_code" 2>/dev/null
  return 0
}

autoload -U add-zsh-hook
add-zsh-hook chpwd __dirvana_hook

# Run on startup only if stdin is a terminal
[[ -t 0 ]] && __dirvana_hook`, binPath, binPath)

	default: // bash
		return fmt.Sprintf(`__dirvana_hook() {
  # Check if Dirvana is disabled
  if [[ "${DIRVANA_ENABLED:-true}" == "false" ]]; then
    return 0
  fi

  # Check if dirvana command exists
  if ! command -v %s &> /dev/null; then
    return 0
  fi

  # Don't run if stdin is not the terminal (prevents TUI interference)
  if [[ ! -t 0 ]]; then
    return 0
  fi

  # Only run if directory changed
  if [[ "$PWD" != "${DIRVANA_PREV_DIR:-}" ]]; then
    local shell_code
    shell_code=$(%s export --prev "${DIRVANA_PREV_DIR:-}")
    local exit_code=$?
    export DIRVANA_PREV_DIR="$PWD"
    [[ $exit_code -eq 0 && -n "$shell_code" ]] && eval "$shell_code" 2>/dev/null
  fi
  return 0
}

# Add to PROMPT_COMMAND
if [[ -z "${PROMPT_COMMAND}" ]]; then
  PROMPT_COMMAND="__dirvana_hook"
elif [[ ! "${PROMPT_COMMAND}" =~ __dirvana_hook ]]; then
  PROMPT_COMMAND="__dirvana_hook;${PROMPT_COMMAND}"
fi

# Run on startup only if stdin is a terminal
[[ -t 0 ]] && __dirvana_hook`, binPath, binPath)
	}
}

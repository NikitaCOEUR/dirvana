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
	// ShellPowerShell represents PowerShell
	ShellPowerShell = "powershell"
	// ShellPwsh represents PowerShell Core (pwsh)
	ShellPwsh = "pwsh"
)

// DetectShell determines the shell type based on the flag or environment.
func DetectShell(shellFlag string) string {
	if shellFlag != "auto" {
		return shellFlag
	}

	// Detect from SHELL env var
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return ShellZsh
	}
	if strings.Contains(shell, "bash") {
		return ShellBash
	}

	// Detect PowerShell on Windows
	psModulePath := os.Getenv("PSModulePath")
	if psModulePath != "" {
		// Check if it's PowerShell Core (pwsh) or Windows PowerShell
		if strings.Contains(psModulePath, "pwsh") {
			return ShellPwsh
		}
		return ShellPowerShell
	}

	// Default to bash
	return ShellBash
}

// GenerateHookCode generates the shell hook code for the specified shell.
func GenerateHookCode(shell string) string {
	// Get the path to the current binary
	binPath, err := os.Executable()
	if err != nil {
		binPath = "dirvana" // Fallback to PATH
	}

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

	case ShellPowerShell, ShellPwsh:
		return fmt.Sprintf(`function __Dirvana-Hook {
    # Check if Dirvana is disabled
    if ($env:DIRVANA_ENABLED -eq "false") {
        return
    }

    # Check if dirvana command exists
    if (-not (Get-Command %s -ErrorAction SilentlyContinue)) {
        return
    }

    $prevDir = $env:DIRVANA_PREV_DIR
    if (-not $prevDir) { $prevDir = "" }

    $shellCode = & %s export --prev $prevDir 2>$null
    $exitCode = $LASTEXITCODE
    $env:DIRVANA_PREV_DIR = $PWD.Path

    if ($exitCode -eq 0 -and $shellCode) {
        Invoke-Expression $shellCode
    }
}

# Hook into location changes
$global:__DirvanaLocationChangedAction = {
    __Dirvana-Hook
}

$null = Register-EngineEvent -SourceIdentifier PowerShell.OnIdle -Action $global:__DirvanaLocationChangedAction

# Run hook on startup
__Dirvana-Hook`, binPath, binPath)

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

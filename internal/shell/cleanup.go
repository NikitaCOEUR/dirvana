package shell

import (
	"strings"
)

// GenerateCleanupCode generates shell code to unset aliases, functions and
// environment variables when leaving a dirvana context.
// shell parameter can be "bash", "zsh", "fish", or "" (generates for all shells)
func GenerateCleanupCode(aliases []string, functions []string, envVars []string, shell string) string {
	var lines []string

	lines = append(lines, "# Dirvana cleanup")
	lines = append(lines, generateAliasCleanup(aliases, shell)...)
	lines = append(lines, generateFunctionCleanup(functions, shell)...)
	lines = append(lines, generateEnvCleanup(envVars, shell)...)

	return strings.Join(lines, "\n") + "\n"
}

// generateAliasCleanup generates shell commands to remove aliases
// Note: We intentionally don't remove completions (complete -r / compdef -d) because:
// - complete -r is very slow in bash (~200ms per call), causing noticeable delay
// - Once the alias is removed, its completion is harmless (never called)
// - This is a performance optimization: instant cleanup vs negligible memory leak
func generateAliasCleanup(aliases []string, shell string) []string {
	if len(aliases) == 0 {
		return nil
	}

	var lines []string
	for _, alias := range aliases {
		if shell == Fish {
			// Fish uses 'functions -e' to remove functions/aliases
			lines = append(lines, "functions -e "+alias+" 2>/dev/null; or true")
		} else {
			// Bash/Zsh use 'unalias'
			lines = append(lines, "unalias "+alias+" 2>/dev/null || true")
		}
	}

	return lines
}

// generateFunctionCleanup generates shell commands to unset functions
func generateFunctionCleanup(functions []string, shell string) []string {
	var lines []string
	for _, fn := range functions {
		if shell == Fish {
			// Fish uses 'functions -e' to remove functions
			lines = append(lines, "functions -e "+fn+" 2>/dev/null; or true")
		} else {
			// Bash/Zsh use 'unset -f'
			lines = append(lines, "unset -f "+fn+" 2>/dev/null || true")
		}
	}
	return lines
}

// generateEnvCleanup generates shell commands to unset environment variables
func generateEnvCleanup(envVars []string, shell string) []string {
	var lines []string
	for _, env := range envVars {
		if shell == Fish {
			// Fish uses 'set -e' to unset variables
			lines = append(lines, "set -e "+env)
		} else {
			// Bash/Zsh use 'unset'
			lines = append(lines, "unset "+env)
		}
	}
	return lines
}

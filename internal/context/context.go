// Package context handles tracking of shell context and environment cleanup.
package context

import (
	"path/filepath"
	"strings"
)

// ShouldCleanup determines if we should clean up the previous context
// Returns true if we're leaving a dirvana context
func ShouldCleanup(previousDir, currentDir string) bool {
	if previousDir == "" {
		return false // No previous context
	}

	if previousDir == currentDir {
		return false // Same directory
	}

	// Check if current is a subdirectory of previous
	relPath, err := filepath.Rel(previousDir, currentDir)
	if err != nil {
		return true // Different contexts
	}

	// If relative path doesn't start with "..", we're in a subdirectory
	isSubdir := !strings.HasPrefix(relPath, "..")

	// Clean up if we're NOT in a subdirectory (we left the context)
	return !isSubdir
}

// GenerateCleanupCode generates shell code to unset variables
func GenerateCleanupCode(aliases []string, functions []string, envVars []string) string {
	var lines []string

	lines = append(lines, "# Dirvana cleanup")

	// Unalias
	for _, alias := range aliases {
		lines = append(lines, "unalias "+alias+" 2>/dev/null || true")
	}

	// Unset functions
	for _, fn := range functions {
		lines = append(lines, "unset -f "+fn+" 2>/dev/null || true")
	}

	// Unset env vars
	for _, env := range envVars {
		lines = append(lines, "unset "+env)
	}

	return strings.Join(lines, "\n") + "\n"
}

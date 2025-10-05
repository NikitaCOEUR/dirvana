// Package context handles tracking of shell context and environment cleanup.
package context

import (
	"path/filepath"
	"strings"
)

// AuthChecker defines the interface for checking directory authorization
type AuthChecker interface {
	IsAllowed(path string) (bool, error)
}

// ConfigProvider defines the interface for finding and checking config files
type ConfigProvider interface {
	FindConfigs(dir string) []string
	IsLocalOnly(dir string) bool
}

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
	lines = append(lines, generateAliasCleanup(aliases)...)
	lines = append(lines, generateFunctionCleanup(functions)...)
	lines = append(lines, generateEnvCleanup(envVars)...)

	return strings.Join(lines, "\n") + "\n"
}

// generateAliasCleanup generates shell commands to remove aliases and their completions
func generateAliasCleanup(aliases []string) []string {
	var lines []string
	for _, alias := range aliases {
		lines = append(lines, "unalias "+alias+" 2>/dev/null || true")
		// Remove bash completion
		lines = append(lines, "complete -r "+alias+" 2>/dev/null || true")
		// Remove zsh completion
		lines = append(lines, "compdef -d "+alias+" 2>/dev/null || true")
	}
	return lines
}

// generateFunctionCleanup generates shell commands to unset functions
func generateFunctionCleanup(functions []string) []string {
	var lines []string
	for _, fn := range functions {
		lines = append(lines, "unset -f "+fn+" 2>/dev/null || true")
	}
	return lines
}

// generateEnvCleanup generates shell commands to unset environment variables
func generateEnvCleanup(envVars []string) []string {
	var lines []string
	for _, env := range envVars {
		lines = append(lines, "unset "+env)
	}
	return lines
}

// GetActiveConfigChain returns the list of directories whose configs should be active
// for the given directory, respecting authorization and local_only flags.
// Returns directories in order from root to leaf.
func GetActiveConfigChain(dir string, auth AuthChecker, configProvider ConfigProvider) []string {
	if dir == "" {
		return []string{}
	}

	// Find all config files in the hierarchy
	configDirs := configProvider.FindConfigs(dir)

	if len(configDirs) == 0 {
		return []string{}
	}

	var activeChain []string
	var localOnlyIndex = -1

	// Process configs from root to leaf
	for i, configDir := range configDirs {
		// Check authorization if auth checker is provided
		if auth != nil {
			allowed, err := auth.IsAllowed(configDir)
			if err != nil || !allowed {
				continue // Skip unauthorized configs
			}
		}

		// Check for local_only flag
		if configProvider.IsLocalOnly(configDir) {
			// When we hit local_only, we need to discard previous configs
			localOnlyIndex = i
			activeChain = []string{configDir}
		} else {
			// Only add if we haven't hit a local_only yet, or we're after it
			if localOnlyIndex == -1 || i > localOnlyIndex {
				activeChain = append(activeChain, configDir)
			}
		}
	}

	return activeChain
}

// CalculateCleanup determines which directories need cleanup when moving
// from prevChain to currentChain. Returns directories that were in prevChain
// but are not in currentChain.
func CalculateCleanup(prevChain, currentChain []string) []string {
	// Convert currentChain to a set for O(1) lookup
	currentSet := make(map[string]bool)
	for _, dir := range currentChain {
		currentSet[dir] = true
	}

	// Find directories that need cleanup
	cleanup := []string{} // Initialize as empty slice, not nil
	for _, dir := range prevChain {
		if !currentSet[dir] {
			cleanup = append(cleanup, dir)
		}
	}

	return cleanup
}

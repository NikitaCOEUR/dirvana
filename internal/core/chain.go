// Package core implements the orchestration engine behind the dirvana
// commands: resolving which configs are active for a directory and keeping
// the merged view cached for the export, exec and completion hot paths.
package core

import (
	"github.com/NikitaCOEUR/dirvana/internal/config"
)

// ConfigProvider is the subset of config.Loader used to walk the hierarchy
type ConfigProvider interface {
	FindConfigs(dir string) []string
	IsLocalOnly(dir string) bool
}

// ActiveConfigChain returns the list of directories whose configs should be
// active for the given directory, respecting authorization and local_only
// flags. Returns directories in order from root to leaf.
func ActiveConfigChain(dir string, auth config.AuthChecker, configProvider ConfigProvider) []string {
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
	currentSet := make(map[string]bool, len(currentChain))
	for _, dir := range currentChain {
		currentSet[dir] = true
	}

	// Find directories that need cleanup
	// Pre-allocate with prevChain length capacity (worst case: all need cleanup)
	cleanup := make([]string, 0, len(prevChain))
	for _, dir := range prevChain {
		if !currentSet[dir] {
			cleanup = append(cleanup, dir)
		}
	}

	return cleanup
}

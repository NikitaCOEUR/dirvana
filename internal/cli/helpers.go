package cli

import (
	"fmt"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/shell"
)

// components holds initialized Dirvana components
type components struct {
	auth   *auth.Auth
	cache  *cache.Cache
	config *config.Loader
	shell  *shell.Generator
}

// initializeComponents creates and initializes all required components
func initializeComponents(cachePath, authPath string) (*components, error) {
	authMgr, err := auth.New(authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	cacheStore, err := cache.New(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	return &components{
		auth:   authMgr,
		cache:  cacheStore,
		config: config.New(),
		shell:  shell.NewGenerator(),
	}, nil
}

// keysFromMap extracts sorted keys from a map[string]string
// Pre-allocates slice with exact capacity to avoid reallocation
func keysFromMap(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// keysFromAliasMap extracts keys from a map[string]config.AliasConfig
func keysFromAliasMap(m map[string]config.AliasConfig) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// mergeTwoKeyLists combines two lists of keys into a single pre-allocated slice
func mergeTwoKeyLists(map1, map2 map[string]string) []string {
	keys := make([]string, 0, len(map1)+len(map2))
	for k := range map1 {
		keys = append(keys, k)
	}
	for k := range map2 {
		keys = append(keys, k)
	}
	return keys
}

// buildCommandMap creates a map of alias/function names to their commands
// This is used by dirvana exec to resolve aliases
func buildCommandMap(aliases map[string]config.AliasConfig, functions map[string]string) map[string]string {
	commandMap := make(map[string]string, len(aliases)+len(functions))

	// Add aliases
	for name, aliasConf := range aliases {
		commandMap[name] = aliasConf.Command
	}

	// Add functions (they'll be executed as shell functions)
	// For now, functions are stored but will need special handling in exec
	for name := range functions {
		// Mark functions with a special prefix so we know to handle them differently
		commandMap[name] = "__dirvana_function__" + name
	}

	return commandMap
}

// buildCompletionMap creates a map of alias names to completion commands
// Only includes aliases that have explicit completion configuration
func buildCompletionMap(aliases map[string]config.AliasConfig) map[string]string {
	completionMap := make(map[string]string)

	for name, aliasConf := range aliases {
		// Only add if there's an explicit completion command (string type)
		if aliasConf.Completion != nil {
			if completionCmd, ok := aliasConf.Completion.(string); ok && completionCmd != "" {
				completionMap[name] = completionCmd
			}
		}
	}

	return completionMap
}

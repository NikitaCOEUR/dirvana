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

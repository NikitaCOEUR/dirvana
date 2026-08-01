package cli

import (
	"fmt"
	"maps"
	"slices"

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

// mapKeys returns the keys of a string-keyed map (unsorted)
func mapKeys[V any](m map[string]V) []string {
	return slices.Collect(maps.Keys(m))
}

// mergeTwoKeyLists combines the keys of two maps into a single slice
func mergeTwoKeyLists(map1, map2 map[string]string) []string {
	return slices.Concat(mapKeys(map1), mapKeys(map2))
}

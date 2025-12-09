package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	dircontext "github.com/NikitaCOEUR/dirvana/internal/context"
	"github.com/NikitaCOEUR/dirvana/internal/shell"
	"github.com/NikitaCOEUR/dirvana/internal/trace"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
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

// getMergedAliasConfigs returns the merged aliases and functions for a directory.
// Respects the full config hierarchy including global config, ignore_global, local_only, and authorization.
// Returns nil if no context is found or not authorized.
func getMergedAliasConfigs(currentDir string, cachePath string, authPath string) (aliases map[string]config.AliasConfig, functions map[string]string, err error) {
	// Initialize components
	comps, err := initializeComponents(cachePath, authPath)
	if err != nil {
		return nil, nil, err
	}

	// Load the full config hierarchy with auth
	// This respects global config, ignore_global, local_only, and authorization
	mergedConfig, _, err := comps.config.LoadHierarchyWithAuth(currentDir, comps.auth)
	if err != nil {
		return nil, nil, err
	}

	// Return aliases and functions
	aliases = mergedConfig.GetAliases()
	functions = mergedConfig.Functions

	return aliases, functions, nil
}

// getMergedCommandMaps returns merged CommandMaps and CompletionMaps for a directory.
// Respects the full config hierarchy including global config, ignore_global, local_only, and authorization.
// Returns nil maps if no context is found or not authorized.
func getMergedCommandMaps(currentDir string, cachePath string, authPath string) (commandMap, completionMap map[string]string, err error) {
	ctx := context.Background()
	defer trace.Region(ctx, "getMergedCommandMaps")()

	// FAST PATH: Check cache with TTL first, before loading heavy components (auth, config)
	// This avoids ~24ms of file I/O in the common case where cache is still fresh
	var cacheStore *cache.Cache
	trace.WithRegion(ctx, "cache.New", func() {
		cacheStore, err = cache.New(cachePath)
	})
	if err != nil {
		return nil, nil, err
	}

	if cachedEntry, found := cacheStore.Get(currentDir); found {
		// Quick validation: check version and TTL only (no file I/O)
		if isCacheValidFast(cachedEntry, version.Version) {
			trace.Log(ctx, "cache", "hit-fast")
			return cachedEntry.MergedCommandMap, cachedEntry.MergedCompletionMap, nil
		}
	}

	// SLOW PATH: Cache miss or TTL expired - need full validation
	// Now load auth and config for full hierarchy validation
	var comps *components
	trace.WithRegion(ctx, "initializeComponents", func() {
		comps, err = initializeComponentsWithCache(cachePath, authPath, cacheStore)
	})
	if err != nil {
		return nil, nil, err
	}

	// Try full cache validation (with hash check)
	if cachedEntry, found := comps.cache.Get(currentDir); found {
		var validEntry *cache.Entry
		var isValid bool
		trace.WithRegion(ctx, "validateMergedCache", func() {
			validEntry, isValid = validateMergedCache(cachedEntry, currentDir, comps.config, comps.auth, version.Version)
		})
		if isValid {
			// Cache hit after full validation
			trace.Log(ctx, "cache", "hit-validated")
			return validEntry.MergedCommandMap, validEntry.MergedCompletionMap, nil
		}
		// Cache invalid, fall through to hierarchy load
		trace.Log(ctx, "cache", "invalid")
	} else {
		trace.Log(ctx, "cache", "miss")
	}

	// Cache miss or invalid: Load the full config hierarchy with auth
	// This respects global config, ignore_global, local_only, and authorization
	var mergedConfig *config.Config
	trace.WithRegion(ctx, "LoadHierarchyWithAuth", func() {
		mergedConfig, _, err = comps.config.LoadHierarchyWithAuth(currentDir, comps.auth)
	})
	if err != nil {
		return nil, nil, err
	}

	// Build command maps from the merged config
	aliases := mergedConfig.GetAliases()
	commandMap = buildCommandMap(aliases, mergedConfig.Functions)
	completionMap = buildCompletionMap(aliases)

	return commandMap, completionMap, nil
}

// isCacheValidFast performs quick cache validation without file I/O
// Only checks version and TTL - used for fast path before loading auth/config
func isCacheValidFast(cacheEntry *cache.Entry, appVersion string) bool {
	// Check version compatibility
	if cacheEntry.Version != appVersion {
		return false
	}

	// Check if merged maps exist (backward compatibility with old cache format)
	if cacheEntry.MergedCommandMap == nil {
		return false
	}

	// Check if hierarchy hash exists
	if cacheEntry.HierarchyHash == "" {
		return false
	}

	// Fast path: if cache was validated recently, trust it without recomputing hash
	return time.Since(cacheEntry.Timestamp) < cacheValidationTTL
}

// initializeComponentsWithCache creates components reusing an existing cache instance
func initializeComponentsWithCache(_, authPath string, cacheStore *cache.Cache) (*components, error) {
	authMgr, err := auth.New(authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	return &components{
		auth:   authMgr,
		cache:  cacheStore,
		config: config.New(),
		shell:  shell.NewGenerator(),
	}, nil
}

// computeHierarchyHash computes a composite hash from all configs in the hierarchy
// Returns: hierarchyHash, configPaths, error
// The hierarchyHash is a concatenation of all individual config hashes, making it
// sensitive to changes in any file in the hierarchy
func computeHierarchyHash(configDirs []string, configLoader *config.Loader) (string, []string, error) {
	var hashes []string
	var paths []string

	for _, configDir := range configDirs {
		// Find config file in this directory
		var configPath string
		for _, name := range config.SupportedConfigNames {
			path := filepath.Join(configDir, name)
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			continue
		}

		// Compute hash for this config file
		hash, err := configLoader.Hash(configPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to hash %s: %w", configPath, err)
		}

		hashes = append(hashes, hash)
		paths = append(paths, configPath)
	}

	// Concatenate all hashes with colons
	hierarchyHash := strings.Join(hashes, ":")
	return hierarchyHash, paths, nil
}

// cacheValidationTTL is the time during which we trust the cache without revalidating
// This avoids expensive hash recalculation on every completion keystroke
const cacheValidationTTL = 2 * time.Second

// validateMergedCache checks if a cached merged configuration is still valid
// Returns the cached entry if valid, or nil + false if cache should be invalidated
func validateMergedCache(cacheEntry *cache.Entry, currentDir string, configLoader *config.Loader, authMgr *auth.Auth, appVersion string) (*cache.Entry, bool) {
	// Check version compatibility
	if cacheEntry.Version != appVersion {
		return nil, false
	}

	// Check if merged maps exist (backward compatibility with old cache format)
	if cacheEntry.MergedCommandMap == nil {
		return nil, false
	}

	// Check if hierarchy hash exists
	if cacheEntry.HierarchyHash == "" {
		return nil, false
	}

	// Fast path: if cache was validated recently, trust it without recomputing hash
	// This dramatically speeds up completion (avoids ~20ms of stat calls per keystroke)
	if time.Since(cacheEntry.Timestamp) < cacheValidationTTL {
		return cacheEntry, true
	}

	// Slow path: recompute active config chain to check if it changed
	// Use GetActiveConfigChain from context package
	activeChain := dircontext.GetActiveConfigChain(currentDir, authMgr, configLoader)

	// Compute current hierarchy hash
	currentHash, _, err := computeHierarchyHash(activeChain, configLoader)
	if err != nil {
		// Failed to compute hash, invalidate cache
		return nil, false
	}

	// Compare hashes
	if currentHash != cacheEntry.HierarchyHash {
		return nil, false
	}

	// Cache is valid!
	return cacheEntry, true
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

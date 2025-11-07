package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeComponents(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	comps, err := initializeComponents(cachePath, authPath)
	require.NoError(t, err)
	assert.NotNil(t, comps.auth)
	assert.NotNil(t, comps.cache)
	assert.NotNil(t, comps.config)
	assert.NotNil(t, comps.shell)
}

func TestInitializeComponents_InvalidAuthPath(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create a file where the auth parent directory should be (will cause MkdirAll to fail)
	authParent := filepath.Join(tmpDir, "auth_parent_is_file")
	err := os.WriteFile(authParent, []byte("blocking file"), 0644)
	require.NoError(t, err)

	invalidAuthPath := filepath.Join(authParent, "auth.json")

	_, err = initializeComponents(cachePath, invalidAuthPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestInitializeComponents_InvalidCachePath(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create a directory where the cache file should be (will cause cache.New to fail)
	invalidCachePath := filepath.Join(tmpDir, "cache_is_a_dir")
	err := os.MkdirAll(invalidCachePath, 0755)
	require.NoError(t, err)

	_, err = initializeComponents(invalidCachePath, authPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize cache")
}

func TestKeysFromMap(t *testing.T) {
	m := map[string]string{
		"alias1": "value1",
		"alias2": "value2",
		"alias3": "value3",
	}

	keys := keysFromMap(m)
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "alias1")
	assert.Contains(t, keys, "alias2")
	assert.Contains(t, keys, "alias3")
}

func TestKeysFromMap_Empty(t *testing.T) {
	m := map[string]string{}
	keys := keysFromMap(m)
	assert.Len(t, keys, 0)
	assert.NotNil(t, keys) // Should return empty slice, not nil
}

func TestMergeTwoKeyLists(t *testing.T) {
	map1 := map[string]string{
		"key1": "val1",
		"key2": "val2",
	}
	map2 := map[string]string{
		"key3": "val3",
		"key4": "val4",
	}

	keys := mergeTwoKeyLists(map1, map2)
	assert.Len(t, keys, 4)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")
	assert.Contains(t, keys, "key4")
}

func TestMergeTwoKeyLists_EmptyMaps(t *testing.T) {
	map1 := map[string]string{}
	map2 := map[string]string{}

	keys := mergeTwoKeyLists(map1, map2)
	assert.Len(t, keys, 0)
	assert.NotNil(t, keys)
}

func TestMergeTwoKeyLists_OneEmpty(t *testing.T) {
	map1 := map[string]string{
		"key1": "val1",
	}
	map2 := map[string]string{}

	keys := mergeTwoKeyLists(map1, map2)
	assert.Len(t, keys, 1)
	assert.Contains(t, keys, "key1")
}

func TestBuildCompletionMap(t *testing.T) {
	aliases := map[string]config.AliasConfig{
		"kc": {
			Command:    "kubecolor",
			Completion: "kubectl", // String completion
		},
		"gs": {
			Command:    "git status",
			Completion: nil, // Auto-detect
		},
		"test": {
			Command:    "echo test",
			Completion: false, // Disabled
		},
		"empty": {
			Command:    "echo empty",
			Completion: "", // Empty string
		},
	}

	completionMap := buildCompletionMap(aliases)

	// Should only include "kc" with explicit string completion
	assert.Len(t, completionMap, 1)
	assert.Equal(t, "kubectl", completionMap["kc"])

	// Others should not be in the map
	assert.NotContains(t, completionMap, "gs")
	assert.NotContains(t, completionMap, "test")
	assert.NotContains(t, completionMap, "empty")
}

func TestBuildCompletionMap_Empty(t *testing.T) {
	aliases := map[string]config.AliasConfig{}
	completionMap := buildCompletionMap(aliases)

	assert.Len(t, completionMap, 0)
	assert.NotNil(t, completionMap)
}

func TestComputeHierarchyHash(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hierarchy of config dirs
	rootDir := filepath.Join(tmpDir, "root")
	childDir := filepath.Join(rootDir, "child")
	grandchildDir := filepath.Join(childDir, "grandchild")

	require.NoError(t, os.MkdirAll(rootDir, 0755))
	require.NoError(t, os.MkdirAll(childDir, 0755))
	require.NoError(t, os.MkdirAll(grandchildDir, 0755))

	// Create config files
	rootConfigPath := filepath.Join(rootDir, ".dirvana.yml")
	childConfigPath := filepath.Join(childDir, ".dirvana.yml")

	rootConfig := `aliases:
  root: echo root
`

	require.NoError(t, os.WriteFile(rootConfigPath, []byte(rootConfig), 0644))
	require.NoError(t, os.WriteFile(childConfigPath, []byte(childAliasConfig), 0644))

	// Create config loader
	loader := config.New()

	// Test: compute hash for hierarchy
	configDirs := []string{rootDir, childDir}
	hierarchyHash, paths, err := computeHierarchyHash(configDirs, loader)

	require.NoError(t, err)
	assert.NotEmpty(t, hierarchyHash)
	assert.Len(t, paths, 2)
	assert.Equal(t, rootConfigPath, paths[0])
	assert.Equal(t, childConfigPath, paths[1])

	// Hash should be in format "hash1:hash2"
	assert.Contains(t, hierarchyHash, ":")

	// Test: same hierarchy should produce same hash
	hierarchyHash2, paths2, err := computeHierarchyHash(configDirs, loader)
	require.NoError(t, err)
	assert.Equal(t, hierarchyHash, hierarchyHash2)
	assert.Equal(t, paths, paths2)

	// Test: change a config file, hash should change
	modifiedChildConfig := `aliases:
  child: echo modified
`
	require.NoError(t, os.WriteFile(childConfigPath, []byte(modifiedChildConfig), 0644))

	hierarchyHash3, _, err := computeHierarchyHash(configDirs, loader)
	require.NoError(t, err)
	assert.NotEqual(t, hierarchyHash, hierarchyHash3, "Hash should change when config changes")
}

func TestComputeHierarchyHash_EmptyHierarchy(t *testing.T) {
	loader := config.New()

	// Empty config dirs
	hierarchyHash, paths, err := computeHierarchyHash([]string{}, loader)

	require.NoError(t, err)
	assert.Empty(t, hierarchyHash)
	assert.Empty(t, paths)
}

func TestComputeHierarchyHash_NoConfigFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dirs without config files
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")

	require.NoError(t, os.MkdirAll(dir1, 0755))
	require.NoError(t, os.MkdirAll(dir2, 0755))

	loader := config.New()

	// Should not error, just return empty
	hierarchyHash, paths, err := computeHierarchyHash([]string{dir1, dir2}, loader)

	require.NoError(t, err)
	assert.Empty(t, hierarchyHash)
	assert.Empty(t, paths)
}

func TestComputeHierarchyHash_SingleConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create single config
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	loader := config.New()

	hierarchyHash, paths, err := computeHierarchyHash([]string{tmpDir}, loader)

	require.NoError(t, err)
	assert.NotEmpty(t, hierarchyHash)
	assert.Len(t, paths, 1)
	assert.Equal(t, configPath, paths[0])

	// Hash should NOT contain ":" for single config
	assert.NotContains(t, hierarchyHash, ":")
}

func TestValidateMergedCache_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create config
	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configPath := filepath.Join(configDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	// Initialize components
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(configDir))

	loader := config.New()

	// Load the hierarchy first (this is what happens in production before validateMergedCache is called)
	_, _, err = loader.LoadHierarchyWithAuth(configDir, authMgr)
	require.NoError(t, err)

	// Compute hierarchy hash
	hierarchyHash, hierarchyPaths, err := computeHierarchyHash([]string{configDir}, loader)
	require.NoError(t, err)

	// Create valid cache entry
	cacheEntry := &cache.Entry{
		Path:                configDir,
		Hash:                hierarchyHash,
		Version:             version.Version,
		Timestamp:           time.Now(),
		MergedCommandMap:    map[string]string{"test": "echo test"},
		MergedCompletionMap: map[string]string{},
		HierarchyHash:       hierarchyHash,
		HierarchyPaths:      hierarchyPaths,
	}

	// Validate cache
	validEntry, isValid := validateMergedCache(cacheEntry, configDir, loader, authMgr, version.Version)
	assert.True(t, isValid)
	assert.Equal(t, cacheEntry, validEntry)
}

func TestValidateMergedCache_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	loader := config.New()

	// Create cache entry with wrong version
	cacheEntry := &cache.Entry{
		Path:                configDir,
		Version:             "0.0.1", // Wrong version
		MergedCommandMap:    map[string]string{"test": "echo test"},
		MergedCompletionMap: map[string]string{},
		HierarchyHash:       "somehash",
	}

	// Should be invalid due to version mismatch
	validEntry, isValid := validateMergedCache(cacheEntry, configDir, loader, authMgr, version.Version)
	assert.False(t, isValid)
	assert.Nil(t, validEntry)
}

func TestValidateMergedCache_NoMergedMaps(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	loader := config.New()

	// Create cache entry without merged maps (old format)
	cacheEntry := &cache.Entry{
		Path:             configDir,
		Version:          version.Version,
		MergedCommandMap: nil, // Missing merged map
		HierarchyHash:    "somehash",
	}

	// Should be invalid due to missing merged map
	validEntry, isValid := validateMergedCache(cacheEntry, configDir, loader, authMgr, version.Version)
	assert.False(t, isValid)
	assert.Nil(t, validEntry)
}

func TestValidateMergedCache_NoHierarchyHash(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	loader := config.New()

	// Create cache entry without hierarchy hash
	cacheEntry := &cache.Entry{
		Path:                configDir,
		Version:             version.Version,
		MergedCommandMap:    map[string]string{"test": "echo test"},
		MergedCompletionMap: map[string]string{},
		HierarchyHash:       "", // Missing hierarchy hash
	}

	// Should be invalid due to missing hierarchy hash
	validEntry, isValid := validateMergedCache(cacheEntry, configDir, loader, authMgr, version.Version)
	assert.False(t, isValid)
	assert.Nil(t, validEntry)
}

func TestValidateMergedCache_HierarchyChanged(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create config
	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configPath := filepath.Join(configDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	// Initialize components
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(configDir))

	loader := config.New()

	// Load the hierarchy first
	_, _, err = loader.LoadHierarchyWithAuth(configDir, authMgr)
	require.NoError(t, err)

	// Compute initial hierarchy hash
	hierarchyHash, hierarchyPaths, err := computeHierarchyHash([]string{configDir}, loader)
	require.NoError(t, err)

	// Create cache entry with old hash
	cacheEntry := &cache.Entry{
		Path:                configDir,
		Hash:                hierarchyHash,
		Version:             version.Version,
		Timestamp:           time.Now(),
		MergedCommandMap:    map[string]string{"test": "echo test"},
		MergedCompletionMap: map[string]string{},
		HierarchyHash:       hierarchyHash,
		HierarchyPaths:      hierarchyPaths,
	}

	// Modify config file (this will change the hash)
	modifiedConfig := `aliases:
  test: echo modified
`
	require.NoError(t, os.WriteFile(configPath, []byte(modifiedConfig), 0644))

	// Should be invalid due to hierarchy hash mismatch
	validEntry, isValid := validateMergedCache(cacheEntry, configDir, loader, authMgr, version.Version)
	assert.False(t, isValid)
	assert.Nil(t, validEntry)
}

func TestValidateMergedCache_NewConfigAdded(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create initial config structure
	rootDir := filepath.Join(tmpDir, "root")
	childDir := filepath.Join(rootDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	// Only create child config initially
	childConfigPath := filepath.Join(childDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(childConfigPath, []byte(childAliasConfig), 0644))

	// Initialize components
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(rootDir))
	require.NoError(t, authMgr.Allow(childDir))

	loader := config.New()

	// Load the hierarchy first (with only child config)
	_, _, err = loader.LoadHierarchyWithAuth(childDir, authMgr)
	require.NoError(t, err)

	// Compute hash with only child config
	hierarchyHash, hierarchyPaths, err := computeHierarchyHash([]string{childDir}, loader)
	require.NoError(t, err)

	// Create cache entry
	cacheEntry := &cache.Entry{
		Path:                childDir,
		Version:             version.Version,
		MergedCommandMap:    map[string]string{"child": "echo child"},
		MergedCompletionMap: map[string]string{},
		HierarchyHash:       hierarchyHash,
		HierarchyPaths:      hierarchyPaths,
	}

	// Add parent config (changes hierarchy)
	rootConfigPath := filepath.Join(rootDir, ".dirvana.yml")
	rootConfig := `aliases:
  root: echo root
`
	require.NoError(t, os.WriteFile(rootConfigPath, []byte(rootConfig), 0644))

	// Now the hierarchy includes both root and child
	// Cache should be invalid because hierarchy changed
	validEntry, isValid := validateMergedCache(cacheEntry, childDir, loader, authMgr, version.Version)
	assert.False(t, isValid)
	assert.Nil(t, validEntry)
}

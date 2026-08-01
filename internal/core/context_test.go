package core

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

const testAliasConfig = `aliases:
  test: echo test
`

const childAliasConfig = `aliases:
  child: echo child
`

func TestContext_CommandMap(t *testing.T) {
	aliases := map[string]config.AliasConfig{
		"simple":  {Command: "echo hello"},
		"complex": {Command: "git status"},
	}
	functions := map[string]string{
		"myfunc": "echo func",
	}

	commandMap := NewContext(aliases, functions).CommandMap()

	assert.Equal(t, "echo hello", commandMap["simple"])
	assert.Equal(t, "git status", commandMap["complex"])
	assert.Equal(t, FunctionPrefix+"myfunc", commandMap["myfunc"])
}

func TestContext_CompletionMap(t *testing.T) {
	aliases := map[string]config.AliasConfig{
		"kc": {
			Command:    "kubecolor",
			Completion: "kubectl", // String completion
		},
		"gs": {
			Command:    "git status",
			Completion: nil, // Auto-detect -> uses command
		},
		"test": {
			Command:    "echo test",
			Completion: false, // Disabled -> no entry
		},
		"empty": {
			Command:    "echo empty",
			Completion: "", // Empty string -> uses command
		},
	}

	completionMap := NewContext(aliases, nil).CompletionMap()

	// Should include all except "test" (completion: false)
	assert.Len(t, completionMap, 3)

	// Explicit string completion
	assert.Equal(t, "kubectl", completionMap["kc"])

	// No completion -> uses command
	assert.Equal(t, "git status", completionMap["gs"])

	// Empty string -> uses command
	assert.Equal(t, "echo empty", completionMap["empty"])

	// Disabled -> no entry
	assert.NotContains(t, completionMap, "test")
}

func TestContext_CompletionMap_Empty(t *testing.T) {
	completionMap := NewContext(nil, nil).CompletionMap()
	assert.Len(t, completionMap, 0)
	assert.NotNil(t, completionMap)
}

func TestHierarchyHash(t *testing.T) {
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
	hierarchyHash, paths, err := HierarchyHash(configDirs, loader)

	require.NoError(t, err)
	assert.NotEmpty(t, hierarchyHash)
	assert.Len(t, paths, 2)
	assert.Equal(t, rootConfigPath, paths[0])
	assert.Equal(t, childConfigPath, paths[1])

	// Hash should be in format "hash1:hash2"
	assert.Contains(t, hierarchyHash, ":")

	// Test: same hierarchy should produce same hash
	hierarchyHash2, paths2, err := HierarchyHash(configDirs, loader)
	require.NoError(t, err)
	assert.Equal(t, hierarchyHash, hierarchyHash2)
	assert.Equal(t, paths, paths2)

	// Test: change a config file, hash should change
	modifiedChildConfig := `aliases:
  child: echo modified
`
	require.NoError(t, os.WriteFile(childConfigPath, []byte(modifiedChildConfig), 0644))

	hierarchyHash3, _, err := HierarchyHash(configDirs, loader)
	require.NoError(t, err)
	assert.NotEqual(t, hierarchyHash, hierarchyHash3, "Hash should change when config changes")
}

func TestHierarchyHash_EmptyHierarchy(t *testing.T) {
	loader := config.New()

	hierarchyHash, paths, err := HierarchyHash([]string{}, loader)

	require.NoError(t, err)
	assert.Empty(t, hierarchyHash)
	assert.Empty(t, paths)
}

func TestHierarchyHash_NoConfigFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dirs without config files
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")

	require.NoError(t, os.MkdirAll(dir1, 0755))
	require.NoError(t, os.MkdirAll(dir2, 0755))

	loader := config.New()

	// Should not error, just return empty
	hierarchyHash, paths, err := HierarchyHash([]string{dir1, dir2}, loader)

	require.NoError(t, err)
	assert.Empty(t, hierarchyHash)
	assert.Empty(t, paths)
}

func TestHierarchyHash_SingleConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create single config
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	loader := config.New()

	hierarchyHash, paths, err := HierarchyHash([]string{tmpDir}, loader)

	require.NoError(t, err)
	assert.NotEmpty(t, hierarchyHash)
	assert.Len(t, paths, 1)
	assert.Equal(t, configPath, paths[0])

	// Hash should NOT contain ":" for single config
	assert.NotContains(t, hierarchyHash, ":")
}

func TestValidateEntry_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create config
	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configPath := filepath.Join(configDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(configDir))

	loader := config.New()

	hierarchyHash, hierarchyPaths, err := HierarchyHash([]string{configDir}, loader)
	require.NoError(t, err)

	entry := &cache.Entry{
		Path:            configDir,
		Hash:            hierarchyHash,
		Version:         version.Version,
		Timestamp:       time.Now(),
		MergedAliases:   map[string]config.AliasConfig{"test": {Command: "echo test"}},
		MergedFunctions: map[string]string{},
		HierarchyHash:   hierarchyHash,
		HierarchyPaths:  hierarchyPaths,
	}

	assert.True(t, validateEntry(entry, configDir, loader, authMgr))
}

func TestValidateEntry_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	loader := config.New()

	entry := &cache.Entry{
		Path:            configDir,
		Version:         "0.0.1", // Wrong version
		MergedAliases:   map[string]config.AliasConfig{"test": {Command: "echo test"}},
		MergedFunctions: map[string]string{},
		HierarchyHash:   "somehash",
	}

	assert.False(t, validateEntry(entry, configDir, loader, authMgr))
}

func TestValidateEntry_NoMergedSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	loader := config.New()

	// Entry without merged snapshot (old format)
	entry := &cache.Entry{
		Path:          configDir,
		Version:       version.Version,
		MergedAliases: nil, // Missing merged snapshot
		HierarchyHash: "somehash",
	}

	assert.False(t, validateEntry(entry, configDir, loader, authMgr))
}

func TestValidateEntry_NoHierarchyHash(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	loader := config.New()

	entry := &cache.Entry{
		Path:            configDir,
		Version:         version.Version,
		MergedAliases:   map[string]config.AliasConfig{"test": {Command: "echo test"}},
		MergedFunctions: map[string]string{},
		HierarchyHash:   "", // Missing hierarchy hash
	}

	assert.False(t, validateEntry(entry, configDir, loader, authMgr))
}

func TestValidateEntry_HierarchyChanged(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create config
	configDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configPath := filepath.Join(configDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(configDir))

	loader := config.New()

	hierarchyHash, hierarchyPaths, err := HierarchyHash([]string{configDir}, loader)
	require.NoError(t, err)

	entry := &cache.Entry{
		Path:            configDir,
		Hash:            hierarchyHash,
		Version:         version.Version,
		Timestamp:       time.Now(),
		MergedAliases:   map[string]config.AliasConfig{"test": {Command: "echo test"}},
		MergedFunctions: map[string]string{},
		HierarchyHash:   hierarchyHash,
		HierarchyPaths:  hierarchyPaths,
	}

	// Modify config file (this will change the hash)
	modifiedConfig := `aliases:
  test: echo modified
`
	require.NoError(t, os.WriteFile(configPath, []byte(modifiedConfig), 0644))

	assert.False(t, validateEntry(entry, configDir, loader, authMgr))
}

func TestValidateEntry_NewConfigAdded(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create initial config structure
	rootDir := filepath.Join(tmpDir, "root")
	childDir := filepath.Join(rootDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	// Only create child config initially
	childConfigPath := filepath.Join(childDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(childConfigPath, []byte(childAliasConfig), 0644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(rootDir))
	require.NoError(t, authMgr.Allow(childDir))

	loader := config.New()

	hierarchyHash, hierarchyPaths, err := HierarchyHash([]string{childDir}, loader)
	require.NoError(t, err)

	entry := &cache.Entry{
		Path:            childDir,
		Version:         version.Version,
		MergedAliases:   map[string]config.AliasConfig{"child": {Command: "echo child"}},
		MergedFunctions: map[string]string{},
		HierarchyHash:   hierarchyHash,
		HierarchyPaths:  hierarchyPaths,
	}

	// Add parent config (changes hierarchy)
	rootConfigPath := filepath.Join(rootDir, ".dirvana.yml")
	rootConfig := `aliases:
  root: echo root
`
	require.NoError(t, os.WriteFile(rootConfigPath, []byte(rootConfig), 0644))

	// Now the hierarchy includes both root and child:
	// the entry must be invalid because the hierarchy changed
	assert.False(t, validateEntry(entry, childDir, loader, authMgr))
}

func TestEngine_Load_FastPathAndInvalidation(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	configPath := filepath.Join(projectDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(projectDir))

	engine := NewEngine(cachePath, authPath)

	// First load: slow path, then written back to cache
	ctx1, err := engine.Load(projectDir)
	require.NoError(t, err)
	assert.Equal(t, "echo test", ctx1.Aliases["test"].Command)

	// The write-back must land in the cache with a merged snapshot
	cacheStore, err := cache.New(cachePath)
	require.NoError(t, err)
	entry, found := cacheStore.Get(projectDir)
	require.True(t, found, "slow path should write the merged snapshot back")
	assert.NotNil(t, entry.MergedAliases)

	// Second load within the TTL: fast path returns the same context
	ctx2, err := engine.Load(projectDir)
	require.NoError(t, err)
	assert.Equal(t, ctx1.Aliases, ctx2.Aliases)

	// exec and completion must see the same resolved commands
	assert.Equal(t, ctx2.CommandMap()["test"], ctx1.Aliases["test"].Command)
}

func TestEngine_Load_SeesConfigEditsAfterTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	configPath := filepath.Join(projectDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(projectDir))

	engine := NewEngine(cachePath, authPath)

	ctx1, err := engine.Load(projectDir)
	require.NoError(t, err)
	assert.Equal(t, "echo test", ctx1.Aliases["test"].Command)

	// Edit the config, then expire the cached entry's TTL manually so the
	// test doesn't have to sleep through cacheValidationTTL
	require.NoError(t, os.WriteFile(configPath, []byte("aliases:\n  test: echo edited\n"), 0644))

	cacheStore, err := cache.New(cachePath)
	require.NoError(t, err)
	entry, found := cacheStore.Get(projectDir)
	require.True(t, found)
	entry.Timestamp = time.Now().Add(-2 * cacheValidationTTL)
	require.NoError(t, cacheStore.Set(entry))

	ctx2, err := engine.Load(projectDir)
	require.NoError(t, err)
	assert.Equal(t, "echo edited", ctx2.Aliases["test"].Command,
		"config edits must be visible once the TTL expired")
}

func TestEngine_Load_NoContext(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0755))

	ctx, err := NewEngine(cachePath, authPath).Load(emptyDir)
	require.NoError(t, err)
	assert.True(t, ctx.Empty())
}

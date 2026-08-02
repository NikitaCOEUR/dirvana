package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cacheMergedConfig stores two different things in one entry: the merged
// snapshot every directory needs for fast exec/completion, and the cleanup
// bookkeeping only directories owning a config need.

func TestCacheMergedConfig_WithLocalConfig(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `aliases:
  test: echo test
env:
  TEST_VAR: value
functions:
  test_func: |
    echo "test function"
`)
	env.allow(t)

	comps, err := initializeComponents(env.CachePath, env.AuthPath)
	require.NoError(t, err)

	cfg, err := comps.config.Load(filepath.Join(env.Dir, ".dirvana.yml"))
	require.NoError(t, err)
	dctx := core.NewContext(cfg.GetAliases(), cfg.Functions)

	cacheMergedConfig(env.Dir, "test_hash", []string{env.Dir}, cfg, dctx, comps, testLogger())

	entry, found := comps.cache.Get(env.Dir)
	require.True(t, found)

	// Cleanup bookkeeping: this directory owns the definitions
	assert.Contains(t, entry.Aliases, "test")
	assert.Contains(t, entry.EnvVars, "TEST_VAR")
	assert.Contains(t, entry.Functions, "test_func")

	// Merged snapshot for the exec/completion fast path
	assert.NotNil(t, entry.MergedAliases)
	assert.NotNil(t, entry.MergedFunctions)
	assert.Equal(t, "test_hash", entry.HierarchyHash)
}

func TestCacheMergedConfig_WithoutLocalConfig(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  test: echo test\nenv:\n  TEST_VAR: value\n")
	env.allow(t)

	// A subdirectory that only inherits: nothing of its own to clean up
	subDir := filepath.Join(env.Dir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	comps, err := initializeComponents(env.CachePath, env.AuthPath)
	require.NoError(t, err)

	cfg, err := comps.config.Load(filepath.Join(env.Dir, ".dirvana.yml"))
	require.NoError(t, err)
	dctx := core.NewContext(cfg.GetAliases(), cfg.Functions)

	hierarchyPaths := []string{env.Dir}
	cacheMergedConfig(subDir, "test_hash", hierarchyPaths, cfg, dctx, comps, testLogger())

	entry, found := comps.cache.Get(subDir)
	require.True(t, found)

	// No cleanup data: unsetting inherited definitions is the owner's job
	assert.Nil(t, entry.Aliases)
	assert.Nil(t, entry.EnvVars)
	assert.Nil(t, entry.Functions)

	// The merged snapshot is still cached, for speed
	assert.NotNil(t, entry.MergedAliases)
	assert.NotNil(t, entry.MergedFunctions)
	assert.Equal(t, "test_hash", entry.HierarchyHash)
	assert.Equal(t, hierarchyPaths, entry.HierarchyPaths)
}

func TestCacheMergedConfig_EmptyHash(t *testing.T) {
	env := newTestEnv(t)

	comps, err := initializeComponents(env.CachePath, env.AuthPath)
	require.NoError(t, err)

	// Without a hierarchy hash there is nothing to invalidate against, so
	// nothing may be cached
	cacheMergedConfig(env.Dir, "", []string{}, nil, nil, comps, testLogger())

	_, found := comps.cache.Get(env.Dir)
	assert.False(t, found)
}

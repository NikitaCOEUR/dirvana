package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/stretchr/testify/require"
)

func TestClean_All(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create cache with some entries
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	// Add some entries
	err = c.Set(&cache.Entry{Path: "/path/one", Hash: "hash1", ShellCode: "code1"})
	require.NoError(t, err)
	err = c.Set(&cache.Entry{Path: "/path/two", Hash: "hash2", ShellCode: "code2"})
	require.NoError(t, err)

	// Clean all
	params := CleanParams{
		CachePath: cachePath,
		LogLevel:  "error",
		All:       true,
	}
	err = Clean(params)
	require.NoError(t, err)

	// Verify cache is empty
	c, err = cache.New(cachePath)
	require.NoError(t, err)
	_, found := c.Get("/path/one")
	require.False(t, found)
	_, found = c.Get("/path/two")
	require.False(t, found)
}

func TestClean_Hierarchy(t *testing.T) {
	// Resolve symlinks for macOS compatibility where /tmp -> /private/tmp
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create test directory structure
	testDir := filepath.Join(tmpDir, "test")
	subDir := filepath.Join(testDir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Create cache with entries in hierarchy
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	// Add entries for different paths
	err = c.Set(&cache.Entry{Path: testDir, Hash: "hash1", ShellCode: "code1"})
	require.NoError(t, err)
	err = c.Set(&cache.Entry{Path: subDir, Hash: "hash2", ShellCode: "code2"})
	require.NoError(t, err)
	err = c.Set(&cache.Entry{Path: "/other/path", Hash: "hash3", ShellCode: "code3"})
	require.NoError(t, err)

	// Change to test directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	err = os.Chdir(testDir)
	require.NoError(t, err)

	// Clean hierarchy
	params := CleanParams{
		CachePath: cachePath,
		LogLevel:  "error",
		All:       false,
	}
	err = Clean(params)
	require.NoError(t, err)

	// Verify hierarchy entries are removed but other path remains
	c, err = cache.New(cachePath)
	require.NoError(t, err)
	_, found := c.Get(testDir)
	require.False(t, found)
	_, found = c.Get(subDir)
	require.False(t, found)
	_, found = c.Get("/other/path")
	require.True(t, found)
}

func TestClean_InvalidCachePath(t *testing.T) {
	params := CleanParams{
		CachePath: "/invalid/path/that/does/not/exist/cache.json",
		LogLevel:  "error",
		All:       true,
	}
	err := Clean(params)
	require.Error(t, err)
}

func TestClean_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Clean empty cache should not error
	params := CleanParams{
		CachePath: cachePath,
		LogLevel:  "error",
		All:       true,
	}
	err := Clean(params)
	require.NoError(t, err)
}

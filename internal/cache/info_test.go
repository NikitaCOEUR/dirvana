package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetCacheInfo_NoCache tests when cache file doesn't exist
func TestGetCacheInfo_NoCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent.json")

	result, err := GetCacheInfo(cachePath)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return info with path but no size/entries
	assert.Equal(t, cachePath, result.Path)
	assert.Equal(t, int64(0), result.Size)
	assert.Equal(t, 0, result.TotalEntries)
}

// TestGetCacheInfo_InvalidJSON tests with invalid JSON
func TestGetCacheInfo_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Write invalid JSON
	err := os.WriteFile(cachePath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	result, err := GetCacheInfo(cachePath)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return partial info (path and size, but no entries count)
	assert.Equal(t, cachePath, result.Path)
	assert.Greater(t, result.Size, int64(0))
	assert.Equal(t, 0, result.TotalEntries) // Invalid JSON means no entries
}

// TestGetCacheInfo_EmptyJSON tests with empty JSON object
func TestGetCacheInfo_EmptyJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	err := os.WriteFile(cachePath, []byte("{}"), 0644)
	require.NoError(t, err)

	result, err := GetCacheInfo(cachePath)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, cachePath, result.Path)
	assert.Equal(t, 0, result.TotalEntries)
}

// TestGetCacheInfo_WithEntries tests with valid cache entries
func TestGetCacheInfo_WithEntries(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cacheContent := `{
		"/path/1": {"hash": "abc123"},
		"/path/2": {"hash": "def456"},
		"/path/3": {"hash": "ghi789"}
	}`
	err := os.WriteFile(cachePath, []byte(cacheContent), 0644)
	require.NoError(t, err)

	result, err := GetCacheInfo(cachePath)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, cachePath, result.Path)
	assert.Greater(t, result.Size, int64(0))
	assert.Equal(t, 3, result.TotalEntries)
}

// TestGetCacheInfo_JSONArray tests with JSON array instead of object
func TestGetCacheInfo_JSONArray(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Write JSON array (invalid format for cache)
	err := os.WriteFile(cachePath, []byte("[1, 2, 3]"), 0644)
	require.NoError(t, err)

	result, err := GetCacheInfo(cachePath)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return partial info
	assert.Equal(t, cachePath, result.Path)
	assert.Greater(t, result.Size, int64(0))
	assert.Equal(t, 0, result.TotalEntries) // Can't parse as map
}

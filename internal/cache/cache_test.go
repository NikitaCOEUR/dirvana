package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	c, err := New(cachePath)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestCache_Set(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	c, err := New(cachePath)
	require.NoError(t, err)

	entry := &Entry{
		Path:      "/test/path",
		Hash:      "abc123",
		ShellCode: "export TEST=1",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		LocalOnly: false,
	}

	err = c.Set(entry)
	require.NoError(t, err)

	// Verify it was set
	got, found := c.Get("/test/path")
	assert.True(t, found)
	assert.Equal(t, entry.Hash, got.Hash)
	assert.Equal(t, entry.ShellCode, got.ShellCode)
}

func TestCache_Get(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	c, err := New(cachePath)
	require.NoError(t, err)

	// Non-existent key
	_, found := c.Get("/non/existent")
	assert.False(t, found)

	// Existing key
	entry := &Entry{
		Path: "/test/path",
		Hash: "abc123",
	}
	require.NoError(t, c.Set(entry))

	got, found := c.Get("/test/path")
	assert.True(t, found)
	assert.Equal(t, "abc123", got.Hash)
}

func TestCache_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	c, err := New(cachePath)
	require.NoError(t, err)

	entry := &Entry{Path: "/test/path", Hash: "abc123"}
	require.NoError(t, c.Set(entry))

	err = c.Delete("/test/path")
	require.NoError(t, err)

	_, found := c.Get("/test/path")
	assert.False(t, found)
}

func TestCache_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create cache and add entry
	c1, err := New(cachePath)
	require.NoError(t, err)

	entry := &Entry{
		Path:      "/test/path",
		Hash:      "abc123",
		ShellCode: "export TEST=1",
		Version:   "1.0.0",
	}
	require.NoError(t, c1.Set(entry))

	// Create new cache instance from same file
	c2, err := New(cachePath)
	require.NoError(t, err)

	got, found := c2.Get("/test/path")
	assert.True(t, found)
	assert.Equal(t, entry.Hash, got.Hash)
	assert.Equal(t, entry.ShellCode, got.ShellCode)
}

func TestCache_InvalidPath(t *testing.T) {
	invalidPath := filepath.Join("/nonexistent", "path", "cache.json")
	_, err := New(invalidPath)
	assert.Error(t, err)
}

func TestCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	c, err := New(cachePath)
	require.NoError(t, err)

	// Add multiple entries
	require.NoError(t, c.Set(&Entry{Path: "/path1", Hash: "hash1"}))
	require.NoError(t, c.Set(&Entry{Path: "/path2", Hash: "hash2"}))

	err = c.Clear()
	require.NoError(t, err)

	_, found1 := c.Get("/path1")
	_, found2 := c.Get("/path2")
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestCache_IsValid(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	c, err := New(cachePath)
	require.NoError(t, err)

	entry := &Entry{
		Path:      "/test/path",
		Hash:      "abc123",
		ShellCode: "export TEST=1",
		Version:   "1.0.0",
		Timestamp: time.Now(),
	}
	require.NoError(t, c.Set(entry))

	// Same hash should be valid
	valid := c.IsValid("/test/path", "abc123", "1.0.0")
	assert.True(t, valid)

	// Different hash should be invalid
	valid = c.IsValid("/test/path", "different", "1.0.0")
	assert.False(t, valid)

	// Different version should be invalid
	valid = c.IsValid("/test/path", "abc123", "2.0.0")
	assert.False(t, valid)

	// Non-existent path should be invalid
	valid = c.IsValid("/non/existent", "abc123", "1.0.0")
	assert.False(t, valid)
}

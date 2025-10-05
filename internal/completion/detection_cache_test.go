package completion

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectionCache_GetSet(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "detection.json")

	cache, err := NewDetectionCache(cachePath)
	require.NoError(t, err)

	// Initially empty
	assert.Equal(t, "", cache.Get("kubectl"))

	// Set and get
	cache.Set("kubectl", "Cobra")
	assert.Equal(t, "Cobra", cache.Get("kubectl"))

	// Save and reload
	err = cache.Save()
	require.NoError(t, err)

	cache2, err := NewDetectionCache(cachePath)
	require.NoError(t, err)
	assert.Equal(t, "Cobra", cache2.Get("kubectl"))
}

func TestDetectionCache_TTL(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "detection.json")

	cache, err := NewDetectionCache(cachePath)
	require.NoError(t, err)

	// Override TTL to 1 millisecond for testing
	cache.ttl = 1 * time.Millisecond

	cache.Set("kubectl", "Cobra")
	assert.Equal(t, "Cobra", cache.Get("kubectl"))

	// Wait for expiry
	time.Sleep(2 * time.Millisecond)

	// Should be expired now
	assert.Equal(t, "", cache.Get("kubectl"))
}

func TestDetectionCache_InvalidPath(t *testing.T) {
	cache, err := NewDetectionCache("/nonexistent/path/cache.json")
	// Should create cache even if file doesn't exist
	require.NoError(t, err)
	assert.NotNil(t, cache)

	// But saving should fail
	cache.Set("test", "value")
	err = cache.Save()
	assert.Error(t, err)
}

func TestDetectionCache_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "empty.json")

	// Create empty file
	err := os.WriteFile(cachePath, []byte(""), 0644)
	require.NoError(t, err)

	// Should handle empty file gracefully
	_, err = NewDetectionCache(cachePath)
	assert.Error(t, err) // JSON unmarshal will fail on empty file
}

func TestDetectionCache_SaveMultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "detection.json")

	cache, err := NewDetectionCache(cachePath)
	require.NoError(t, err)

	// Save multiple times
	cache.Set("tool1", "Cobra")
	err = cache.Save()
	require.NoError(t, err)

	cache.Set("tool2", "UrfaveCli")
	err = cache.Save()
	require.NoError(t, err)

	cache.Set("tool3", "BashComplete")
	err = cache.Save()
	require.NoError(t, err)

	// Reload and verify all entries
	cache2, err := NewDetectionCache(cachePath)
	require.NoError(t, err)
	assert.Equal(t, "Cobra", cache2.Get("tool1"))
	assert.Equal(t, "UrfaveCli", cache2.Get("tool2"))
	assert.Equal(t, "BashComplete", cache2.Get("tool3"))
}

func TestDetectionCache_OverwriteEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "detection.json")

	cache, err := NewDetectionCache(cachePath)
	require.NoError(t, err)

	// Set initial value
	cache.Set("kubectl", "BashComplete")
	assert.Equal(t, "BashComplete", cache.Get("kubectl"))

	// Overwrite with new value
	cache.Set("kubectl", "Cobra")
	assert.Equal(t, "Cobra", cache.Get("kubectl"))

	// Save and reload
	err = cache.Save()
	require.NoError(t, err)

	cache2, err := NewDetectionCache(cachePath)
	require.NoError(t, err)
	assert.Equal(t, "Cobra", cache2.Get("kubectl"))
}

func TestDetectionCache_SaveToReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0555) // Read-only directory
	require.NoError(t, err)
	defer func() { _ = os.Chmod(readOnlyDir, 0755) }() // Restore permissions for cleanup

	cachePath := filepath.Join(readOnlyDir, "cache.json")
	cache, err := NewDetectionCache(cachePath)
	require.NoError(t, err)

	cache.Set("test", "value")
	err = cache.Save()
	assert.Error(t, err, "Should fail to save to read-only directory")
}

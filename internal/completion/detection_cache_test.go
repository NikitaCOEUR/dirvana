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

	cache := NewDetectionCache(cachePath)

	// Initially empty
	assert.Equal(t, "", cache.Get("kubectl"))

	// Set and get
	cache.Set("kubectl", "Cobra")
	assert.Equal(t, "Cobra", cache.Get("kubectl"))

	// Save and reload
	err := cache.Save()
	require.NoError(t, err)

	cache2 := NewDetectionCache(cachePath)
	assert.Equal(t, "Cobra", cache2.Get("kubectl"))
}

func TestDetectionCache_TTL(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "detection.json")

	cache := NewDetectionCache(cachePath)

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
	cache := NewDetectionCache("/nonexistent/path/cache.json")
	// Should create cache even if file doesn't exist
	assert.NotNil(t, cache)

	// But saving should fail
	cache.Set("test", "value")
	err := cache.Save()
	assert.Error(t, err)
}

func TestDetectionCache_CorruptFile(t *testing.T) {
	tmpDir := t.TempDir()

	for name, content := range map[string]string{
		"empty.json":   "",
		"corrupt.json": "{not json",
	} {
		t.Run(name, func(t *testing.T) {
			cachePath := filepath.Join(tmpDir, name)
			require.NoError(t, os.WriteFile(cachePath, []byte(content), 0o644))

			// A corrupt cache file yields a usable empty cache
			cache := NewDetectionCache(cachePath)
			require.NotNil(t, cache)
			assert.Equal(t, "", cache.Get("kubectl"))

			// And the cache is fully functional afterwards
			cache.Set("kubectl", "Cobra")
			require.NoError(t, cache.Save())
			assert.Equal(t, "Cobra", NewDetectionCache(cachePath).Get("kubectl"))
		})
	}
}

func TestDetectionCache_SaveMultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "detection.json")

	cache := NewDetectionCache(cachePath)

	// Save multiple times
	cache.Set("tool1", "Cobra")
	require.NoError(t, cache.Save())

	cache.Set("tool2", "UrfaveCli")
	require.NoError(t, cache.Save())

	cache.Set("tool3", "BashComplete")
	require.NoError(t, cache.Save())

	// Reload and verify all entries
	cache2 := NewDetectionCache(cachePath)
	assert.Equal(t, "Cobra", cache2.Get("tool1"))
	assert.Equal(t, "UrfaveCli", cache2.Get("tool2"))
	assert.Equal(t, "BashComplete", cache2.Get("tool3"))
}

func TestDetectionCache_OverwriteEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "detection.json")

	cache := NewDetectionCache(cachePath)

	// Set initial value
	cache.Set("kubectl", "BashComplete")
	assert.Equal(t, "BashComplete", cache.Get("kubectl"))

	// Overwrite with new value
	cache.Set("kubectl", "Cobra")
	assert.Equal(t, "Cobra", cache.Get("kubectl"))

	// Save and reload
	require.NoError(t, cache.Save())

	cache2 := NewDetectionCache(cachePath)
	assert.Equal(t, "Cobra", cache2.Get("kubectl"))
}

func TestDetectionCache_SaveToReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0o555) // Read-only directory
	require.NoError(t, err)
	defer func() { _ = os.Chmod(readOnlyDir, 0o755) }() // Restore permissions for cleanup

	cachePath := filepath.Join(readOnlyDir, "cache.json")
	cache := NewDetectionCache(cachePath)

	cache.Set("test", "value")
	err = cache.Save()
	assert.Error(t, err, "Should fail to save to read-only directory")
}

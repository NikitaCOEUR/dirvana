package cli

import (
	"path/filepath"
	"testing"

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
	_, err := initializeComponents("/invalid/cache.json", "/root/auth.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize")
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

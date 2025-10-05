package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindCacheEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create cache with entries
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(parentDir, "child")
	grandchildDir := filepath.Join(childDir, "grandchild")

	// Add entry for parent
	err = c.Set(&cache.Entry{
		Path:      parentDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"test": "echo test",
		},
	})
	require.NoError(t, err)

	// Add entry for child
	err = c.Set(&cache.Entry{
		Path:      childDir,
		Hash:      "hash2",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"child": "echo child",
		},
	})
	require.NoError(t, err)

	// Test finding entry in parent dir
	entry, found := findCacheEntry(c, parentDir)
	assert.True(t, found)
	assert.Equal(t, parentDir, entry.Path)
	assert.Equal(t, "echo test", entry.CommandMap["test"])

	// Test finding entry in child dir
	entry, found = findCacheEntry(c, childDir)
	assert.True(t, found)
	assert.Equal(t, childDir, entry.Path)
	assert.Equal(t, "echo child", entry.CommandMap["child"])

	// Test finding parent entry from grandchild dir (walks up)
	entry, found = findCacheEntry(c, grandchildDir)
	assert.True(t, found)
	assert.Equal(t, childDir, entry.Path)

	// Test not finding entry in unrelated dir
	unrelatedDir := filepath.Join(tmpDir, "unrelated")
	entry, found = findCacheEntry(c, unrelatedDir)
	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestExec_NoCacheEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create empty cache
	_, err := cache.New(cachePath)
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Alias:     "test",
		Args:      []string{},
	}

	err = Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no dirvana context found")
}

func TestExec_AliasNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	// Create cache with entry but different alias
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"other": "echo other",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Alias:     "nonexistent",
		Args:      []string{},
	}

	err = Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "alias 'nonexistent' not found")
}

func TestExec_EmptyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	// Create cache with empty command
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"empty": "",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Alias:     "empty",
		Args:      []string{},
	}

	err = Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
}

func TestExec_CommandNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	// Create cache with non-existent command
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"badcmd": "this-command-does-not-exist-anywhere",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Alias:     "badcmd",
		Args:      []string{},
	}

	err = Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command not found")
}

func TestExec_CommandWithArgs(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	// Create cache with echo command (exists on all systems)
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"e": "echo hello",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Alias:     "e",
		Args:      []string{"world"},
	}

	// This will execute "echo hello world"
	// Note: syscall.Exec will replace the test process, so we can't really test this directly
	// The test will verify the setup is correct before exec is called
	_ = params // Params are valid, but we can't actually call Exec in test
}

func TestExec_MultiWordCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	// Create cache with multi-word command
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"ll": "ls -la",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Alias:     "ll",
		Args:      []string{"/tmp"},
	}

	// This would execute "ls -la /tmp"
	// We can't test syscall.Exec directly, but we verify the setup
	_ = params
}

func TestExec_InvalidCachePath(t *testing.T) {
	params := ExecParams{
		CachePath: "/nonexistent/path/cache.json",
		LogLevel:  "error",
		Alias:     "test",
		Args:      []string{},
	}

	err := Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load cache")
}

func TestFindCacheEntry_RootDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create cache
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	// Try to find entry starting from root (should stop at root)
	entry, found := findCacheEntry(c, "/")
	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestFindCacheEntry_CleanPath(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	c, err := cache.New(cachePath)
	require.NoError(t, err)

	dir := filepath.Join(tmpDir, "test")

	// Add entry
	err = c.Set(&cache.Entry{
		Path:      dir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"test": "echo test",
		},
	})
	require.NoError(t, err)

	// Test with path that needs cleaning (has . or ..)
	dirtyPath := filepath.Join(dir, ".", "subdir", "..")
	entry, found := findCacheEntry(c, dirtyPath)
	assert.True(t, found)
	assert.Equal(t, dir, entry.Path)
}

func TestExec_CacheLoadFailure(t *testing.T) {
	// Use a directory path as cache path (will fail to load)
	tmpDir := t.TempDir()
	
	params := ExecParams{
		CachePath: tmpDir, // Directory, not a file
		LogLevel:  "error",
		Alias:     "test",
		Args:      []string{},
	}

	err := Exec(params)
	assert.Error(t, err)
	// Should fail at cache loading
}


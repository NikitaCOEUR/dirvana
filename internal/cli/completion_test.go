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

func TestCompletion_EmptyWords(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{},
		CWord:     0,
	}

	err := Completion(params)
	assert.NoError(t, err)
}

func TestCompletion_NoCacheEntry(t *testing.T) {
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

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"test"},
		CWord:     0,
	}

	err = Completion(params)
	// Should not return error, just no completions
	assert.NoError(t, err)
}

func TestCompletion_AliasNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

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

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"nonexistent"},
		CWord:     0,
	}

	err = Completion(params)
	// Should not return error, just no completions
	assert.NoError(t, err)
}

func TestCompletion_WithCompletionOverride(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create cache with entry that has completion override
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"k": "kubecolor",
		},
		CompletionMap: map[string]string{
			"k": "kubectl", // Use kubectl for completion even though command is kubecolor
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"k", "get"},
		CWord:     1,
	}

	// This will try to execute completion, which will fail since kubectl may not exist
	// But we're testing the logic path
	err = Completion(params)
	// Error is acceptable here since the actual command may not exist
	// We just want to verify the function doesn't panic
	_ = err
}

func TestCompletion_BasicFlow(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create cache with echo command (should exist on all systems)
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"e": "echo",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"e", "test"},
		CWord:     1,
	}

	// Execute completion (may or may not produce output, but should not crash)
	err = Completion(params)
	// Error is acceptable - we're testing the code path
	_ = err
}

func TestCompletion_FunctionAlias(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create cache with function alias
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"myfunc": "__dirvana_function__myfunc",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"myfunc", "arg1"},
		CWord:     1,
	}

	// Functions don't have smart completions
	err = Completion(params)
	assert.NoError(t, err)
}

func TestCompletion_EmptyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

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

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"empty"},
		CWord:     0,
	}

	err = Completion(params)
	assert.NoError(t, err)
}

func TestCompletion_CommandNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create cache with non-existent command
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"notfound": "this-command-does-not-exist-12345",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"notfound", "arg"},
		CWord:     1,
	}

	// Should not error, just no completions
	err = Completion(params)
	assert.NoError(t, err)
}

func TestCompletion_CompletionBeyondLastWord(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create cache with echo command
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"e": "echo",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	// CWord beyond existing words - completing a new word
	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"e", "test"},
		CWord:     2, // Beyond last word
	}

	err = Completion(params)
	// May or may not error depending on completion engine
	_ = err
}

func TestCompletion_WithCurrentWord(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0755))

	// Create cache with echo command
	c, err := cache.New(cachePath)
	require.NoError(t, err)

	err = c.Set(&cache.Entry{
		Path:      workDir,
		Hash:      "hash1",
		Timestamp: time.Now(),
		Version:   version.Version,
		CommandMap: map[string]string{
			"e": "echo",
		},
	})
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	// Test with current word being completed
	params := CompletionParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Words:     []string{"e", "tes"},
		CWord:     1, // Completing "tes"
	}

	err = Completion(params)
	// May or may not error depending on completion engine
	_ = err
}


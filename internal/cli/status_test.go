package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const testAliasConfig = `aliases:
  test: echo test
`

func TestStatus_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error even with no config
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_NotAuthorized(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte(testAliasConfig), 0644)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error but should show not authorized
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_Authorized(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
  ll: ls -la
functions:
  greet: echo "Hello, $1!"
env:
  PROJECT_NAME: myproject
  GIT_BRANCH:
    sh: git branch --show-current
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and display status
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  ll: ls -la
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	// First export to populate cache
	exportParams := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}
	err = Export(exportParams)
	require.NoError(t, err)

	// Now check status - should show cache hit
	statusParams := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}
	err = Status(statusParams)
	require.NoError(t, err)
}

func TestStatus_WithHierarchy(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create parent and child directories
	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(tmpDir, "parent", "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	// Create parent config
	parentConfig := filepath.Join(parentDir, ".dirvana.yml")
	parentContent := `aliases:
  parent: echo parent
`
	require.NoError(t, os.WriteFile(parentConfig, []byte(parentContent), 0644))

	// Create child config
	childConfig := filepath.Join(childDir, ".dirvana.yml")
	childContent := `aliases:
  child: echo child
`
	require.NoError(t, os.WriteFile(childConfig, []byte(childContent), 0644))

	// Authorize both directories
	require.NoError(t, Allow(authPath, parentDir))
	require.NoError(t, Allow(authPath, childDir))

	// Change to child dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(childDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and show hierarchy
	err = Status(params)
	require.NoError(t, err)
}

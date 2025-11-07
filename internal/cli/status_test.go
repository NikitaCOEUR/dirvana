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

const childAliasConfig = `aliases:
  child: echo child
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
	require.NoError(t, os.WriteFile(childConfig, []byte(childAliasConfig), 0644))

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

func TestStatus_WithFlags(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with flags
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
local_only: true
ignore_global: true
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

	// Should succeed and display flags
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithLongAlias(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with a very long alias command
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	longCommand := "echo this is a very long command that should be truncated when displayed in the status output to avoid cluttering the terminal"
	configContent := "aliases:\n  longcmd: " + longCommand + "\n"
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and truncate the long alias
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithAdvancedAliases(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with advanced alias configurations
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  # Simple alias
  simple: echo test

  # Advanced alias with completion disabled
  nocomp:
    command: echo no completion
    completion: false

  # Advanced alias with inherited completion
  withcomp:
    command: kubectl get pods
    completion: kubectl
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

	// Should succeed and handle different alias types
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_InvalidCachePath(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := "/invalid/path/that/does/not/exist/cache.json"
	authPath := filepath.Join(tmpDir, "auth.json")

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte("aliases:\n  test: echo test\n"), 0644)
	require.NoError(t, err)
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{CachePath: cachePath, AuthPath: authPath}
	err = Status(params)
	require.Error(t, err)
}

func TestStatus_InvalidAuthPath(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := "/invalid/path/that/does/not/exist/auth.json"

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	params := StatusParams{CachePath: cachePath, AuthPath: authPath}
	err = Status(params)
	require.Error(t, err)
}

func TestStatus_WithMixedAuthorizations(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create directory hierarchy: A/B/C
	dirA := filepath.Join(tmpDir, "A")
	dirB := filepath.Join(dirA, "B")
	dirC := filepath.Join(dirB, "C")
	require.NoError(t, os.MkdirAll(dirC, 0755))

	// Create configs in each directory
	configA := filepath.Join(dirA, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configA, []byte("aliases:\n  a: echo a\n"), 0644))

	configB := filepath.Join(dirB, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configB, []byte("aliases:\n  b: echo b\n"), 0644))

	configC := filepath.Join(dirC, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configC, []byte("aliases:\n  c: echo c\n"), 0644))

	// Authorize only A and C, not B
	require.NoError(t, Allow(authPath, dirA))
	require.NoError(t, Allow(authPath, dirC))

	// Change to dirC
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(dirC))

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should show authorization status for each config
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithLocalOnly(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create directory hierarchy: parent/child
	dirParent := filepath.Join(tmpDir, "parent")
	dirChild := filepath.Join(dirParent, "child")
	require.NoError(t, os.MkdirAll(dirChild, 0755))

	// Create parent config
	configParent := filepath.Join(dirParent, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configParent, []byte("aliases:\n  parent: echo parent\n"), 0644))

	// Create child config with local_only
	configChild := filepath.Join(dirChild, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configChild, []byte("local_only: true\naliases:\n  child: echo child\n"), 0644))

	// Authorize both directories
	require.NoError(t, Allow(authPath, dirParent))
	require.NoError(t, Allow(authPath, dirChild))

	// Change to dirChild
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(dirChild))

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should only show child config due to local_only
	err = Status(params)
	require.NoError(t, err)
}

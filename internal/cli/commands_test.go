package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPathConst = "/test/path"

func TestAllow(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	testPath := testPathConst

	err := Allow(authPath, testPath)
	require.NoError(t, err)

	// Verify it was actually allowed
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	allowed, err := authMgr.IsAllowed(testPath)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAllow_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Test with empty path - auth package allows it, so no error
	err := Allow(authPath, "")
	require.NoError(t, err)
}

func TestAllow_InvalidAuthPath(t *testing.T) {
	// Test with invalid auth path
	err := Allow("/invalid/nonexistent/dir/auth.json", "/test/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestRevoke(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	testPath := testPathConst

	// First allow
	err := Allow(authPath, testPath)
	require.NoError(t, err)

	// Then revoke
	err = Revoke(authPath, testPath)
	require.NoError(t, err)

	// Verify it was revoked
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	allowed, err := authMgr.IsAllowed(testPath)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRevoke_NotAuthorized(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	testPath := testPathConst

	// Try to revoke a path that was never authorized
	err := Revoke(authPath, testPath)
	// Should not error even if path wasn't authorized
	require.NoError(t, err)
}

func TestRevoke_InvalidAuthPath(t *testing.T) {
	// Test with invalid auth path
	err := Revoke("/invalid/nonexistent/dir/auth.json", "/test/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Test with no authorized paths
	err := List(authPath)
	require.NoError(t, err)

	// Add some paths
	err = Allow(authPath, "/test/path1")
	require.NoError(t, err)
	err = Allow(authPath, "/test/path2")
	require.NoError(t, err)

	// Test with authorized paths
	err = List(authPath)
	require.NoError(t, err)
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run init
	err = Init()
	require.NoError(t, err)

	// Verify config file was created
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "yaml-language-server: $schema=")
	assert.Contains(t, content, "aliases:")
	assert.Contains(t, content, "functions:")
	assert.Contains(t, content, "env:")
	assert.Contains(t, content, "local_only:")
}

func TestInit_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create config file first
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Run init should fail
	err = Init()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestExport_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error even with no config
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_NotAuthorized(t *testing.T) {
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
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error but should warn (we just test it doesn't crash)
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_Authorized(t *testing.T) {
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
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and generate shell code
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_CacheHit(t *testing.T) {
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

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// First call - should generate and cache
	err = Export(params)
	require.NoError(t, err)

	// Second call - should use cache
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_WithContextCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create parent and child directories
	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(tmpDir, "child")
	require.NoError(t, os.MkdirAll(parentDir, 0755))
	require.NoError(t, os.MkdirAll(childDir, 0755))

	// Create configs
	parentConfig := filepath.Join(parentDir, ".dirvana.yml")
	parentContent := `aliases:
  parent: echo parent
`
	require.NoError(t, os.WriteFile(parentConfig, []byte(parentContent), 0644))

	childConfig := filepath.Join(childDir, ".dirvana.yml")
	childContent := `aliases:
  child: echo child
`
	require.NoError(t, os.WriteFile(childConfig, []byte(childContent), 0644))

	// Authorize both directories
	require.NoError(t, Allow(authPath, parentDir))
	require.NoError(t, Allow(authPath, childDir))

	// Change to parent dir and export
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(parentDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	err = Export(params)
	require.NoError(t, err)

	// Change to child dir with previous dir set - should trigger cleanup
	err = os.Chdir(childDir)
	require.NoError(t, err)

	params.PrevDir = parentDir
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_WithShellEnv(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with shell-based env vars
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
env:
  STATIC_VAR: static
  GIT_BRANCH:
    sh: git branch --show-current
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and generate shell code with env vars
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_WithFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with functions
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `functions:
  greet: |
    echo "Hello, $1!"
  mkcd: |
    mkdir -p "$1" && cd "$1"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and generate shell code with functions
	err = Export(params)
	require.NoError(t, err)
}

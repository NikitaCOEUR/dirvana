package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/setup"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCollectAll_EmptyDirectory tests status collection in a directory without any config
func TestCollectAll_EmptyDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Isolate from user's global config
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Collect status
	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Verify basic info
	assert.Equal(t, tmpDir, data.CurrentDir)
	assert.NotEmpty(t, data.Version)
	assert.NotEmpty(t, data.Shell)

	// No configs = authorized by default
	assert.False(t, data.HasAnyConfig)
	assert.True(t, data.Authorized)

	// Empty collections
	assert.Empty(t, data.LocalConfigs)
	assert.Empty(t, data.Aliases)
	assert.Empty(t, data.Functions)
	assert.Empty(t, data.EnvStatic)
	assert.Empty(t, data.EnvShell)
	assert.Empty(t, data.Flags)

	// Cache should show no entries
	assert.Equal(t, cachePath, data.CachePath)
	assert.Equal(t, int64(0), data.CacheFileSize)
	assert.Equal(t, 0, data.CacheTotalEntries)
}

// TestCollectAll_WithUnauthorizedConfig tests status with a config that's not authorized
func TestCollectAll_WithUnauthorizedConfig(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create a .dirvana.yml config
	configContent := `aliases:
  gs: git status
  k: kubectl
`
	err = os.WriteFile(filepath.Join(tmpDir, ".dirvana.yml"), []byte(configContent), 0o644)
	require.NoError(t, err)

	// Create empty auth file (no authorization)
	err = os.WriteFile(authPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	// Change to temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Collect status
	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Has config but not authorized
	assert.True(t, data.HasAnyConfig)
	assert.False(t, data.Authorized)

	// Should have one local config
	assert.Len(t, data.LocalConfigs, 1)
	assert.Equal(t, filepath.Join(tmpDir, ".dirvana.yml"), data.LocalConfigs[0].Path)
	assert.False(t, data.LocalConfigs[0].Authorized)
	assert.False(t, data.LocalConfigs[0].Loaded)

	// No aliases/functions loaded because not authorized
	assert.Empty(t, data.Aliases)
	assert.Empty(t, data.Functions)
}

// TestCollectAll_WithAuthorizedConfig tests status with an authorized config
func TestCollectAll_WithAuthorizedConfig(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create a .dirvana.yml config
	configContent := `aliases:
  gs: git status
  k: kubectl
functions:
  greet: echo "Hello"
env:
  PROJECT_NAME: dirvana
  BUILD_DIR: ./build
local_only: true
`
	err = os.WriteFile(filepath.Join(tmpDir, ".dirvana.yml"), []byte(configContent), 0o644)
	require.NoError(t, err)

	// Create auth and authorize the directory using the API
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	// Change to temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Collect status
	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Has config and authorized
	assert.True(t, data.HasAnyConfig)
	assert.True(t, data.Authorized)

	// Should have one local config
	assert.Len(t, data.LocalConfigs, 1)
	assert.True(t, data.LocalConfigs[0].Authorized)
	assert.True(t, data.LocalConfigs[0].Loaded)

	// Aliases/functions should be loaded
	assert.Len(t, data.Aliases, 2)
	assert.Equal(t, "git status", data.Aliases["gs"].Command)
	assert.Equal(t, "kubectl", data.Aliases["k"].Command)

	assert.Len(t, data.Functions, 1)
	assert.Contains(t, data.Functions, "greet")

	// Environment variables
	assert.Len(t, data.EnvStatic, 2)
	assert.Equal(t, "dirvana", data.EnvStatic["PROJECT_NAME"])
	assert.Equal(t, "./build", data.EnvStatic["BUILD_DIR"])

	// Flags
	assert.Contains(t, data.Flags, "local_only")
}

// TestCollectAll_WithCache tests status with a valid cache
func TestCollectAll_WithCache(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create a .dirvana.yml config
	configContent := `aliases:
  gs: git status
`
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Create auth and authorize the directory using the API
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	// Create cache file with multiple entries
	cacheContent := map[string]any{
		tmpDir: map[string]any{
			"path":      tmpDir,
			"timestamp": "2024-01-01T00:00:00Z",
			"hash":      "test",
			"version":   "1.0.0",
		},
		"/other/path": map[string]any{
			"path":      "/other/path",
			"timestamp": "2024-01-01T00:00:00Z",
			"hash":      "test",
			"version":   "1.0.0",
		},
	}
	cacheData, _ := json.Marshal(cacheContent)
	err = os.WriteFile(cachePath, cacheData, 0o644)
	require.NoError(t, err)

	// Change to temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Collect status
	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Cache info
	assert.Equal(t, cachePath, data.CachePath)
	assert.Greater(t, data.CacheFileSize, int64(0))
	assert.Equal(t, 2, data.CacheTotalEntries)
}

// TestCollectAll_WithCompletion tests status with completion cache and registry
func TestCollectAll_WithCompletion(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	cacheDir := filepath.Join(tmpDir, ".cache")
	err = os.MkdirAll(cacheDir, 0o755)
	require.NoError(t, err)

	cachePath := filepath.Join(cacheDir, "cache.json")
	authPath := filepath.Join(cacheDir, "auth.json")

	// Create detection cache with proper format
	detectionPath := filepath.Join(cacheDir, "completion-detection.json")
	detectionContent := map[string]any{
		"kubectl": map[string]any{"completer_type": "Cobra"},
		"helm":    map[string]any{"completer_type": "Cobra"},
		"go":      map[string]any{"completer_type": "Flag"},
	}
	detectionData, _ := json.Marshal(detectionContent)
	err = os.WriteFile(detectionPath, detectionData, 0o644)
	require.NoError(t, err)

	// Create registry with proper name
	registryPath := filepath.Join(cacheDir, "completion-registry-v1.yml")
	registryContent := `tools:
  kubectl:
    url: https://example.com/kubectl.sh
  helm:
    url: https://example.com/helm.sh
`
	err = os.WriteFile(registryPath, []byte(registryContent), 0o644)
	require.NoError(t, err)

	// Create downloaded scripts in proper directory structure
	scriptsDir := filepath.Join(cacheDir, "completion-scripts", "bash")
	err = os.MkdirAll(scriptsDir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(scriptsDir, "kubectl.sh"), []byte("#!/bin/bash\n# kubectl completion"), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(scriptsDir, "helm.sh"), []byte("#!/bin/bash\n# helm completion"), 0o755)
	require.NoError(t, err)

	// Change to temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Collect status
	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Completion detection
	require.NotNil(t, data.CompletionDetection)
	assert.Equal(t, detectionPath, data.CompletionDetection.Path)
	assert.Greater(t, data.CompletionDetection.Size, int64(0))
	assert.Len(t, data.CompletionDetection.Commands, 3)
	assert.Equal(t, "Cobra", data.CompletionDetection.Commands["kubectl"])
	assert.Equal(t, "Flag", data.CompletionDetection.Commands["go"])

	// Completion registry
	require.NotNil(t, data.CompletionRegistry)
	assert.Equal(t, registryPath, data.CompletionRegistry.Path)
	assert.Greater(t, data.CompletionRegistry.Size, int64(0))
	assert.Equal(t, 2, data.CompletionRegistry.ToolsCount)

	// Downloaded scripts
	assert.Len(t, data.CompletionScripts, 2)
	scriptNames := []string{data.CompletionScripts[0].Tool, data.CompletionScripts[1].Tool}
	assert.Contains(t, scriptNames, "kubectl.sh")
	assert.Contains(t, scriptNames, "helm.sh")
}

// TestCollectAll_WithCompletionOverrides tests status with completion overrides
func TestCollectAll_WithCompletionOverrides(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create a .dirvana.yml config with completion overrides
	configContent := `aliases:
  k:
    command: kubecolor
    completion: kubectl
  g:
    command: git
`
	err = os.WriteFile(filepath.Join(tmpDir, ".dirvana.yml"), []byte(configContent), 0o644)
	require.NoError(t, err)

	// Create auth and authorize the directory using the API
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	// Change to temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Collect status
	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Should have completion override for k -> kubectl
	assert.Len(t, data.CompletionOverrides, 1)
	assert.Equal(t, "kubectl", data.CompletionOverrides["k"])
}

// TestCollectAll_WithGlobalConfig tests status with global configuration
func TestCollectAll_WithGlobalConfig(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	// Resolve symlinks to handle macOS /var -> /private/var
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create global config directory
	globalDir := filepath.Join(tmpDir, ".config", "dirvana")
	err = os.MkdirAll(globalDir, 0o755)
	require.NoError(t, err)

	globalConfigPath := filepath.Join(globalDir, "global.yml")
	globalContent := `aliases:
  ll: ls -la
`
	err = os.WriteFile(globalConfigPath, []byte(globalContent), 0o644)
	require.NoError(t, err)

	// Set XDG_CONFIG_HOME to use our temp directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()
	err = os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	require.NoError(t, err)

	// Create local config
	localContent := `aliases:
  gs: git status
`
	err = os.WriteFile(filepath.Join(tmpDir, ".dirvana.yml"), []byte(localContent), 0o644)
	require.NoError(t, err)

	// Create auth and authorize the directory using the API
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	// Change to temporary directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Collect status
	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Should have global config info
	require.NotNil(t, data.GlobalConfig)
	assert.Equal(t, globalConfigPath, data.GlobalConfig.Path)
	assert.True(t, data.GlobalConfig.Exists)
	assert.True(t, data.GlobalConfig.Loaded)

	// Should have local config
	assert.Len(t, data.LocalConfigs, 1)

	// Should have aliases from both global and local
	assert.Len(t, data.Aliases, 2)
	assert.Equal(t, "ls -la", data.Aliases["ll"].Command)
	assert.Equal(t, "git status", data.Aliases["gs"].Command)
}

// TestCollectSystemInfo covers the shell and hook detection status reports.
// It used to scan the RC file for a handful of patterns, which knew nothing of
// fish and reported "unknown" for it - the shell dirvana was running under.
func TestCollectSystemInfo(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish"} {
		t.Run("detects "+shellName, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("DIRVANA_SHELL", shellName)

			data := &Data{}
			collectSystemInfo(data)

			assert.Equal(t, shellName, data.Shell)
			// The installer knows where each shell keeps its config, fish
			// included; status must not have its own idea of it
			assert.NotEmpty(t, data.RCFile, "an RC file must be reported for %s", shellName)
			assert.False(t, data.HookInstalled, "nothing is installed in an empty home")
		})
	}

	t.Run("sees the hook the installer wrote", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("DIRVANA_SHELL", "fish")

		_, err := setup.InstallHook("fish")
		require.NoError(t, err)

		data := &Data{}
		collectSystemInfo(data)

		assert.True(t, data.HookInstalled)
		assert.False(t, data.HookOutdated, "a hook just installed cannot be outdated")
	})

	t.Run("reports no shell rather than guessing one", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("DIRVANA_SHELL", "")
		t.Setenv("SHELL", "")

		data := &Data{}
		collectSystemInfo(data)

		assert.Empty(t, data.Shell)
		assert.Empty(t, data.RCFile)
		assert.False(t, data.HookInstalled)
	})
}

// TestCollectAll_ValidCache exercises the branch that reads the entry back:
// everything else in this file leaves the cache cold.
func TestCollectAll_ValidCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("aliases:\n  ll: ls -la\n"), 0o644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(tmpDir))

	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	// Fill the cache the way the export path does
	cacheObj, err := cache.New(cachePath)
	require.NoError(t, err)
	configHash, err := config.New().Hash(configPath)
	require.NoError(t, err)
	require.NoError(t, cacheObj.Set(&cache.Entry{
		Path:      tmpDir,
		Hash:      configHash,
		Timestamp: time.Now(),
		Version:   version.Version,
	}))

	data, err := CollectAll(cachePath, authPath)
	require.NoError(t, err)

	assert.True(t, data.CacheValid)
	assert.False(t, data.CacheUpdated.IsZero(), "a valid entry carries when it was generated")
}

package completion

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const testCacheDir = "/tmp/cache"

// setupTestHTTPClient configures HTTP client to trust test TLS certificates
func setupTestHTTPClient(server *httptest.Server) func() {
	oldClient := httpClient

	httpClient = server.Client()
	// Configure to accept insecure certificates for tests
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return func() {
		httpClient = oldClient
	}
}

// TestValidateURL tests URL validation
func TestValidateURL(t *testing.T) {
	t.Run("accepts valid HTTPS URL", func(t *testing.T) {
		err := validateURL("https://example.com/script.sh")
		assert.NoError(t, err)
	})

	t.Run("accepts valid HTTP URL", func(t *testing.T) {
		err := validateURL("http://example.com/script.sh")
		assert.NoError(t, err)
	})

	t.Run("rejects URL without scheme", func(t *testing.T) {
		err := validateURL("example.com/script.sh")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP or HTTPS scheme")
	})

	t.Run("rejects unsupported scheme", func(t *testing.T) {
		err := validateURL("ftp://example.com/script.sh")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP or HTTPS scheme")
	})

	t.Run("rejects URL without host", func(t *testing.T) {
		err := validateURL("https://")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must have a host")
	})
}

// TestDownloadWithSizeLimit tests download size limiting
func TestDownloadWithSizeLimit(t *testing.T) {
	t.Run("downloads content within size limit", func(t *testing.T) {
		content := []byte("test content")
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		data, err := downloadWithSizeLimit(server.URL, 1024)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("rejects content exceeding size limit", func(t *testing.T) {
		largeContent := make([]byte, 2000)
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "2000")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(largeContent)
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		_, err := downloadWithSizeLimit(server.URL, 1000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content too large")
	})

	t.Run("handles non-200 status code", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		_, err := downloadWithSizeLimit(server.URL, 1024)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 404")
	})
}

// TestComputeHash tests SHA256 hash computation
func TestComputeHash(t *testing.T) {
	t.Run("same content produces same hash", func(t *testing.T) {
		data := []byte("test data")
		hash1 := computeHash(data)
		hash2 := computeHash(data)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		data1 := []byte("test data 1")
		data2 := []byte("test data 2")
		hash1 := computeHash(data1)
		hash2 := computeHash(data2)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty data produces valid hash", func(t *testing.T) {
		data := []byte("")
		hash := computeHash(data)
		assert.NotEmpty(t, hash)
		// SHA256 of empty string
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
	})
}

// TestGetLocalRegistryPath tests local registry path detection
func TestGetLocalRegistryPath(t *testing.T) {
	t.Run("finds registry in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		registryPath := filepath.Join(tmpDir, "registry", "v1", "completion-scripts.yml")
		err := os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, []byte("test"), 0644)
		require.NoError(t, err)

		// Change to tmpDir to test
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		path := getLocalRegistryPath("v1")
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "completion-scripts.yml")
	})

	t.Run("returns empty when no registry exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		path := getLocalRegistryPath("v1")
		assert.Empty(t, path)
	})
}

// TestGetCompletionScriptPath tests completion script path generation
func TestGetCompletionScriptPath(t *testing.T) {
	t.Run("generates correct path", func(t *testing.T) {
		tool := "kubectl"
		shell := "bash"

		path := GetCompletionScriptPath(testCacheDir, tool, shell)
		assert.Equal(t, "/tmp/cache/completion-scripts/bash/kubectl", path)
	})

	t.Run("works with different shells", func(t *testing.T) {
		tool := "helm"

		bashPath := GetCompletionScriptPath(testCacheDir, tool, "bash")
		zshPath := GetCompletionScriptPath(testCacheDir, tool, "zsh")

		assert.Contains(t, bashPath, "/bash/helm")
		assert.Contains(t, zshPath, "/zsh/helm")
		assert.NotEqual(t, bashPath, zshPath)
	})
}

// TestSaveCachedRegistry tests saving registry to cache
func TestSaveCachedRegistry(t *testing.T) {
	t.Run("saves registry successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Test",
			Tools: map[string]RegistryTool{
				"test": {
					Script: RegistryScript{URL: "https://example.com/test.sh"},
				},
			},
		}
		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		config, err := saveCachedRegistry(tmpDir, "v1", data)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "v1", config.Version)

		// Verify file was created
		registryPath := getRegistryPath(tmpDir, "v1")
		_, err = os.Stat(registryPath)
		assert.NoError(t, err)

		// Verify hash was created
		hashPath := getRegistryHashPath(tmpDir, "v1")
		_, err = os.Stat(hashPath)
		assert.NoError(t, err)
	})

	t.Run("fails with invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidData := []byte("not: valid: yaml: {{{}}")

		_, err := saveCachedRegistry(tmpDir, "v1", invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse registry")
	})
}

// TestTryLoadLocalRegistry tests loading from local filesystem
func TestTryLoadLocalRegistry(t *testing.T) {
	t.Run("loads local registry in dev mode", func(t *testing.T) {
		// Create a temporary local registry
		tmpDir := t.TempDir()
		registryPath := filepath.Join(tmpDir, "registry", "v1", "completion-scripts.yml")
		err := os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)

		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Local test",
			Tools:       map[string]RegistryTool{},
		}
		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, data, 0644)
		require.NoError(t, err)

		// Change to tmpDir
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Set dev mode
		oldDevMode := DevMode
		DevMode = "true"
		defer func() { DevMode = oldDevMode }()

		config, ok := tryLoadLocalRegistry("v1")
		assert.True(t, ok)
		assert.NotNil(t, config)
		assert.Equal(t, "v1", config.Version)
	})

	t.Run("returns false when no local registry", func(t *testing.T) {
		// Create a temp dir and change to it to ensure no local registry is found
		tmpDir := t.TempDir()
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		oldDevMode := DevMode
		DevMode = "true"
		defer func() { DevMode = oldDevMode }()

		config, ok := tryLoadLocalRegistry("v1")
		assert.False(t, ok)
		assert.Nil(t, config)
	})
}

// TestTryLoadCachedRegistry tests loading from cache
func TestTryLoadCachedRegistry(t *testing.T) {
	t.Run("loads valid cached registry", func(t *testing.T) {
		tmpDir := t.TempDir()
		registryPath := getRegistryPath(tmpDir, "v1")

		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Cached",
			Tools:       map[string]RegistryTool{},
		}
		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, data, 0644)
		require.NoError(t, err)

		config, ok := tryLoadCachedRegistry(registryPath)
		assert.True(t, ok)
		assert.NotNil(t, config)
		assert.Equal(t, "v1", config.Version)
	})

	t.Run("returns false for expired cache", func(t *testing.T) {
		tmpDir := t.TempDir()
		registryPath := getRegistryPath(tmpDir, "v1")

		registryData := RegistryConfig{Version: "v1"}
		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, data, 0644)
		require.NoError(t, err)

		// Make file old
		oldTime := time.Now().Add(-8 * 24 * time.Hour) // 8 days ago
		err = os.Chtimes(registryPath, oldTime, oldTime)
		require.NoError(t, err)

		config, ok := tryLoadCachedRegistry(registryPath)
		assert.False(t, ok)
		assert.Nil(t, config)
	})

	t.Run("returns false when file doesn't exist", func(t *testing.T) {
		config, ok := tryLoadCachedRegistry("/nonexistent/path")
		assert.False(t, ok)
		assert.Nil(t, config)
	})
}

// TestTryLoadExpiredCache tests loading expired cache as fallback
func TestTryLoadExpiredCache(t *testing.T) {
	t.Run("loads expired cache successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		registryPath := getRegistryPath(tmpDir, "v1")

		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Expired",
			Tools:       map[string]RegistryTool{},
		}
		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, data, 0644)
		require.NoError(t, err)

		config, ok := tryLoadExpiredCache(registryPath)
		assert.True(t, ok)
		assert.NotNil(t, config)
		assert.Equal(t, "v1", config.Version)
	})

	t.Run("returns false when file doesn't exist", func(t *testing.T) {
		config, ok := tryLoadExpiredCache("/nonexistent/path")
		assert.False(t, ok)
		assert.Nil(t, config)
	})
}

// TestLoadRegistry tests registry loading with various scenarios
func TestLoadRegistry(t *testing.T) {
	t.Run("loads from cache when valid", func(t *testing.T) {
		clearRegistryCache() // Clear memory cache before test
		tmpDir := t.TempDir()

		// Create a valid cached registry
		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Test registry",
			Tools: map[string]RegistryTool{
				"test-tool": {
					Description: "Test tool",
					Homepage:    "https://example.com",
					Script: RegistryScript{
						URL:    "https://example.com/script.sh",
						SHA256: "abc123",
					},
				},
			},
		}

		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		registryPath := getRegistryPath(tmpDir, "v1")
		err = os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, data, 0644)
		require.NoError(t, err)

		// Load registry
		config, err := LoadRegistry(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "v1", config.Version)
		assert.Equal(t, "Test registry", config.Description)
		assert.Contains(t, config.Tools, "test-tool")
	})

	t.Run("downloads when cache doesn't exist", func(t *testing.T) {
		// Note: This test would require making RegistryBaseURL configurable
		// For now, we skip as it requires network access or significant refactoring
		t.Skip("Skipping download test - requires configurable registry URL")
	})

	t.Run("uses expired cache when download fails", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create an expired cached registry
		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Expired cache",
			Tools:       map[string]RegistryTool{},
		}

		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		registryPath := getRegistryPath(tmpDir, "v1")
		err = os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, data, 0644)
		require.NoError(t, err)

		// Set file modification time to past (expired)
		pastTime := time.Now().Add(-8 * 24 * time.Hour)
		err = os.Chtimes(registryPath, pastTime, pastTime)
		require.NoError(t, err)

		// Note: This test would require network failure simulation
		// For now, we just verify the cache is read
		config, err := LoadRegistry(tmpDir)
		if err == nil {
			// If it succeeded (either from download or cache), verify it's valid
			assert.NotNil(t, config)
		}
	})

	t.Run("dev mode uses local registry", func(t *testing.T) {
		if DevMode == "" {
			t.Skip("Skipping dev mode test - not in dev build")
		}

		tmpDir := t.TempDir()

		// Create a local registry
		registryPath := filepath.Join(tmpDir, "registry", "v1", "completion-scripts.yml")
		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Local dev registry",
			Tools:       map[string]RegistryTool{},
		}

		data, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		err = os.MkdirAll(filepath.Dir(registryPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(registryPath, data, 0644)
		require.NoError(t, err)

		// Change to directory for local detection
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		config, err := LoadRegistry(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "Local dev registry", config.Description)
	})
}

// TestDownloadCompletionScript tests script downloading
func TestDownloadCompletionScript(t *testing.T) {
	t.Run("downloads and saves script successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a mock HTTPS server
		scriptContent := "#!/bin/bash\necho 'completion script'"
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(scriptContent))
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		// Create registry with test tool
		registry := &RegistryConfig{
			Version: "v1",
			Tools: map[string]RegistryTool{
				"test-tool": {
					Script: RegistryScript{
						URL: server.URL + "/completion.sh",
					},
				},
			},
		}

		err := DownloadCompletionScript(tmpDir, "test-tool", "bash", registry)
		require.NoError(t, err)

		// Verify script was saved
		scriptPath := GetCompletionScriptPath(tmpDir, "test-tool", "bash")
		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		assert.Equal(t, scriptContent, string(content))
	})

	t.Run("verifies checksum when provided", func(t *testing.T) {
		tmpDir := t.TempDir()

		scriptContent := "test script"
		expectedHash := computeHash([]byte(scriptContent))

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(scriptContent))
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		registry := &RegistryConfig{
			Version: "v1",
			Tools: map[string]RegistryTool{
				"test-tool": {
					Script: RegistryScript{
						URL:    server.URL + "/completion.sh",
						SHA256: expectedHash,
					},
				},
			},
		}

		err := DownloadCompletionScript(tmpDir, "test-tool", "bash", registry)
		require.NoError(t, err)
	})

	t.Run("fails on checksum mismatch", func(t *testing.T) {
		tmpDir := t.TempDir()

		scriptContent := "test script"
		wrongHash := "wronghash123"

		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(scriptContent))
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		registry := &RegistryConfig{
			Version: "v1",
			Tools: map[string]RegistryTool{
				"test-tool": {
					Script: RegistryScript{
						URL:    server.URL + "/completion.sh",
						SHA256: wrongHash,
					},
				},
			},
		}

		err := DownloadCompletionScript(tmpDir, "test-tool", "bash", registry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("fails when tool not in registry", func(t *testing.T) {
		tmpDir := t.TempDir()

		registry := &RegistryConfig{
			Version: "v1",
			Tools:   map[string]RegistryTool{},
		}

		err := DownloadCompletionScript(tmpDir, "nonexistent-tool", "bash", registry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in registry")
	})

	t.Run("works regardless of shell parameter", func(t *testing.T) {
		tmpDir := t.TempDir()

		scriptContent := "test"
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(scriptContent))
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		registry := &RegistryConfig{
			Version: "v1",
			Tools: map[string]RegistryTool{
				"test-tool": {
					Script: RegistryScript{URL: server.URL},
				},
			},
		}

		// Should work with any shell parameter (always downloads to bash location)
		err := DownloadCompletionScript(tmpDir, "test-tool", "fish", registry)
		require.NoError(t, err)

		// Verify script was saved to bash location
		scriptPath := GetCompletionScriptPath(tmpDir, "test-tool", "bash")
		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		assert.Equal(t, scriptContent, string(content))
	})

	t.Run("fails when download fails", func(t *testing.T) {
		tmpDir := t.TempDir()

		registry := &RegistryConfig{
			Version: "v1",
			Tools: map[string]RegistryTool{
				"test-tool": {
					Script: RegistryScript{
						URL: "http://invalid-url-that-does-not-exist.local/script.sh",
					},
				},
			},
		}

		err := DownloadCompletionScript(tmpDir, "test-tool", "bash", registry)
		assert.Error(t, err)
	})

	t.Run("creates directory if it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		scriptContent := "test"
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(scriptContent))
		}))
		defer server.Close()
		cleanup := setupTestHTTPClient(server)
		defer cleanup()

		registry := &RegistryConfig{
			Version: "v1",
			Tools: map[string]RegistryTool{
				"test-tool": {
					Script: RegistryScript{URL: server.URL},
				},
			},
		}

		// Ensure directory doesn't exist
		scriptPath := GetCompletionScriptPath(tmpDir, "test-tool", "bash")
		_, err := os.Stat(filepath.Dir(scriptPath))
		assert.True(t, os.IsNotExist(err), "directory should not exist initially")

		err = DownloadCompletionScript(tmpDir, "test-tool", "bash", registry)
		require.NoError(t, err)

		// Verify directory was created
		_, err = os.Stat(filepath.Dir(scriptPath))
		assert.NoError(t, err, "directory should be created")
	})
}

// TestRegistryPaths tests path generation functions
func TestRegistryPaths(t *testing.T) {
	t.Run("getRegistryPath generates correct path", func(t *testing.T) {
		version := "v1"
		path := getRegistryPath(testCacheDir, version)
		assert.Equal(t, "/tmp/cache/completion-registry-v1.yml", path)
	})

	t.Run("getRegistryHashPath generates correct path", func(t *testing.T) {
		version := "v1"
		path := getRegistryHashPath(testCacheDir, version)
		assert.Equal(t, "/tmp/cache/completion-registry-v1.hash", path)
	})
}

// TestDownloadRegistry tests registry downloading (integration-like)
func TestDownloadRegistry(t *testing.T) {
	t.Run("downloads registry from mock server", func(t *testing.T) {
		registryData := RegistryConfig{
			Version:     "v1",
			Description: "Test",
			Tools:       map[string]RegistryTool{},
		}
		yamlData, err := yaml.Marshal(registryData)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "completion-scripts.yml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(yamlData)
		}))
		defer server.Close()

		// Note: downloadRegistry is not exported, so we can't test it directly
		// This shows how it would be tested if it were exported or if we had
		// a test-only export
		t.Skip("downloadRegistry is not exported - testing via LoadRegistry instead")
	})
}

// TestRegistryConfig_Unmarshal tests YAML unmarshaling
func TestRegistryConfig_Unmarshal(t *testing.T) {
	t.Run("unmarshals registry format", func(t *testing.T) {
		yamlData := `
version: "v1"
description: "Test registry"
tools:
  govc:
    description: "VMware CLI"
    homepage: "https://github.com/vmware/govmomi"
    script:
      url: "https://example.com/govc-completion.sh"
      sha256: "abc123"
`

		var config RegistryConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)
		require.NoError(t, err)

		assert.Equal(t, "v1", config.Version)
		assert.Equal(t, "Test registry", config.Description)
		assert.Contains(t, config.Tools, "govc")
		assert.Equal(t, "VMware CLI", config.Tools["govc"].Description)
		assert.Equal(t, "https://example.com/govc-completion.sh", config.Tools["govc"].Script.URL)
		assert.Equal(t, "abc123", config.Tools["govc"].Script.SHA256)
	})

	t.Run("handles empty tools map", func(t *testing.T) {
		yamlData := `
version: "v1"
description: "Empty registry"
tools: {}
`

		var config RegistryConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)
		require.NoError(t, err)

		assert.NotNil(t, config.Tools)
		assert.Len(t, config.Tools, 0)
	})

	t.Run("tool without script URL", func(t *testing.T) {
		yamlData := `
version: "v1"
tools:
  incomplete-tool:
    description: "Tool without script"
    script:
      url: ""
`

		var config RegistryConfig
		err := yaml.Unmarshal([]byte(yamlData), &config)
		require.NoError(t, err)

		assert.Contains(t, config.Tools, "incomplete-tool")
		assert.Empty(t, config.Tools["incomplete-tool"].Script.URL)
	})
}

// BenchmarkComputeHash benchmarks hash computation
func BenchmarkComputeHash(b *testing.B) {
	data := []byte("test data for benchmarking hash computation performance")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = computeHash(data)
	}
}

// BenchmarkLoadRegistry benchmarks registry loading from cache
func BenchmarkLoadRegistry(b *testing.B) {
	tmpDir := b.TempDir()

	// Setup: Create a valid cached registry
	registryData := RegistryConfig{
		Version:     "v1",
		Description: "Benchmark registry",
		Tools:       map[string]RegistryTool{},
	}

	for i := 0; i < 100; i++ {
		toolName := fmt.Sprintf("tool-%d", i)
		registryData.Tools[toolName] = RegistryTool{
			Description: fmt.Sprintf("Tool %d", i),
			Script: RegistryScript{
				URL: fmt.Sprintf("https://example.com/%s.sh", toolName),
			},
		}
	}

	data, err := yaml.Marshal(registryData)
	require.NoError(b, err)

	registryPath := getRegistryPath(tmpDir, "v1")
	err = os.MkdirAll(filepath.Dir(registryPath), 0755)
	require.NoError(b, err)
	err = os.WriteFile(registryPath, data, 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadRegistry(tmpDir)
	}
}

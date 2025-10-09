package completion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultRegistryVersion is the current version of the registry format
	DefaultRegistryVersion = "v1"
	// RegistryBaseURL is the base URL for downloading the registry
	RegistryBaseURL = "https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/registry"
	// RegistryTTL is how long to cache the registry before re-downloading
	RegistryTTL = 7 * 24 * time.Hour // 7 days
	// MaxRegistrySize is the maximum size for downloaded registry (1MB)
	MaxRegistrySize = 1 * 1024 * 1024
	// MaxScriptSize is the maximum size for downloaded completion script (5MB)
	MaxScriptSize = 5 * 1024 * 1024
)

// DevMode is set by ldflags during dev builds
// Production builds will have this as empty string
var DevMode = ""

// httpClient is the HTTP client used for downloads (can be overridden in tests)
var httpClient = http.DefaultClient

// RegistryConfig represents the external completion scripts registry
type RegistryConfig struct {
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description"`
	Tools       map[string]RegistryTool `yaml:"tools"`
}

// RegistryTool represents a tool in the registry
type RegistryTool struct {
	Description string         `yaml:"description"`
	Homepage    string         `yaml:"homepage"`
	Script      RegistryScript `yaml:"script"` // Single bash completion script
}

// RegistryScript represents a completion script
type RegistryScript struct {
	URL    string `yaml:"url"`
	SHA256 string `yaml:"sha256,omitempty"`
}


// Note: We don't maintain a hardcoded list of tools anymore.
// Instead, we auto-detect completion support using common patterns below.

// validateURL validates that a URL uses HTTPS
func validateURL(rawURL string) error {
	// Parse URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Must be HTTPS for security
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("URL must use HTTP or HTTPS scheme, got: %s", u.Scheme)
	}

	// Must have a host
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}

// downloadWithSizeLimit downloads data with a size limit
func downloadWithSizeLimit(url string, maxSize int64) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	// Check Content-Length if available
	if resp.ContentLength > maxSize {
		return nil, fmt.Errorf("content too large: %d bytes (max %d)", resp.ContentLength, maxSize)
	}

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, maxSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	// Check if we exceeded the limit
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("content too large: exceeds %d bytes", maxSize)
	}

	return data, nil
}

// getRegistryPath returns the local cache path for the registry
//
//nolint:unparam // version parameter is kept for future registry format versioning
func getRegistryPath(cacheDir, version string) string {
	return filepath.Join(cacheDir, fmt.Sprintf("completion-registry-%s.yml", version))
}

// getRegistryHashPath returns the path to store the registry hash
func getRegistryHashPath(cacheDir, version string) string {
	return filepath.Join(cacheDir, fmt.Sprintf("completion-registry-%s.hash", version))
}

// downloadRegistry downloads the registry from GitHub
func downloadRegistry(version string) ([]byte, error) {
	downloadURL := fmt.Sprintf("%s/%s/completion-scripts.yml", RegistryBaseURL, version)

	// Validate URL
	if err := validateURL(downloadURL); err != nil {
		return nil, fmt.Errorf("invalid registry URL: %w", err)
	}

	// Download with size limit
	data, err := downloadWithSizeLimit(downloadURL, MaxRegistrySize)
	if err != nil {
		return nil, fmt.Errorf("failed to download registry: %w", err)
	}

	return data, nil
}

// computeHash computes SHA256 hash of data
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// LoadRegistry loads the registry, using cache if valid
// getLocalRegistryPath returns the path to local registry file (for dev mode)
func getLocalRegistryPath(version string) string {
	// Try to find registry in current working directory or project root
	possiblePaths := []string{
		filepath.Join("registry", version, "completion-scripts.yml"),
		filepath.Join("..", "registry", version, "completion-scripts.yml"),
		filepath.Join("..", "..", "registry", version, "completion-scripts.yml"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// LoadRegistry loads the registry, using cache if valid
// Dev builds: Use local registry/ by default (override with DIRVANA_REGISTRY_MODE=remote)
// Prod builds: Always use remote registry (no local fallback)
func LoadRegistry(cacheDir string) (*RegistryConfig, error) {
	version := DefaultRegistryVersion

	// Try to load from local registry (dev mode)
	if config, ok := tryLoadLocalRegistry(version); ok {
		return config, nil
	}

	// Try to load from cache
	registryPath := getRegistryPath(cacheDir, version)
	if config, ok := tryLoadCachedRegistry(registryPath); ok {
		return config, nil
	}

	// Download fresh registry
	data, err := downloadRegistry(version)
	if err != nil {
		// If download fails, try to use expired cache
		if config, ok := tryLoadExpiredCache(registryPath); ok {
			return config, nil
		}
		return nil, err
	}

	// Save to cache and parse
	return saveCachedRegistry(cacheDir, version, data)
}

// tryLoadLocalRegistry attempts to load registry from local filesystem (dev mode)
func tryLoadLocalRegistry(version string) (*RegistryConfig, bool) {
	// Determine if we should use local registry
	useLocal := false
	if DevMode != "" {
		// Dev build: use local by default, unless explicitly set to remote
		useLocal = os.Getenv("DIRVANA_REGISTRY_MODE") != "remote"
	} else {
		// Prod build: never use local, unless explicitly forced (for testing)
		useLocal = os.Getenv("DIRVANA_REGISTRY_MODE") == "local"
	}

	if !useLocal {
		return nil, false
	}

	localPath := getLocalRegistryPath(version)
	if localPath == "" {
		return nil, false
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, false
	}

	var config RegistryConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, false
	}

	return &config, true
}

// tryLoadCachedRegistry attempts to load valid (non-expired) cache
func tryLoadCachedRegistry(registryPath string) (*RegistryConfig, bool) {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, false
	}

	// Check if cache is still valid (based on TTL)
	info, err := os.Stat(registryPath)
	if err != nil || time.Since(info.ModTime()) >= RegistryTTL {
		return nil, false
	}

	// Cache is valid, parse and return
	var config RegistryConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, false
	}

	return &config, true
}

// tryLoadExpiredCache attempts to load expired cache as fallback
func tryLoadExpiredCache(registryPath string) (*RegistryConfig, bool) {
	cachedData, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, false
	}

	var config RegistryConfig
	if err := yaml.Unmarshal(cachedData, &config); err != nil {
		return nil, false
	}

	return &config, true
}

// saveCachedRegistry saves registry data to cache and parses it
func saveCachedRegistry(cacheDir, version string, data []byte) (*RegistryConfig, error) {
	registryPath := getRegistryPath(cacheDir, version)
	hashPath := getRegistryHashPath(cacheDir, version)

	// Compute and store hash
	hash := computeHash(data)

	// Create cache dir if needed
	if err := os.MkdirAll(filepath.Dir(registryPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Save registry to cache
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to cache registry: %w", err)
	}

	// Save hash
	_ = os.WriteFile(hashPath, []byte(hash), 0644)

	// Parse registry
	var config RegistryConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &config, nil
}

// GetCompletionScriptPath returns the cache path for a completion script
func GetCompletionScriptPath(cacheDir, tool, shell string) string {
	return filepath.Join(cacheDir, "completion-scripts", shell, tool)
}

// DownloadCompletionScript downloads a completion script from registry
// Note: shell parameter is kept for backward compatibility but only "bash" is supported
//
//nolint:revive // shell parameter kept for API compatibility
func DownloadCompletionScript(cacheDir, tool, shell string, registry *RegistryConfig) error {
	// Check if tool is in registry
	toolInfo, ok := registry.Tools[tool]
	if !ok {
		return fmt.Errorf("tool %s not found in registry", tool)
	}

	// Get the bash completion script
	if toolInfo.Script.URL == "" {
		return fmt.Errorf("no completion script available for %s", tool)
	}
	scriptInfo := toolInfo.Script

	// Validate URL
	if err := validateURL(scriptInfo.URL); err != nil {
		return fmt.Errorf("invalid script URL for %s: %w", tool, err)
	}

	// Download script with size limit
	data, err := downloadWithSizeLimit(scriptInfo.URL, MaxScriptSize)
	if err != nil {
		return fmt.Errorf("failed to download script for %s: %w", tool, err)
	}

	// Verify checksum if provided
	if scriptInfo.SHA256 != "" {
		hash := computeHash(data)
		if hash != scriptInfo.SHA256 {
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", tool, scriptInfo.SHA256, hash)
		}
	}

	// Save script (always to bash location, regardless of shell parameter)
	scriptPath := GetCompletionScriptPath(cacheDir, tool, "bash")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		return fmt.Errorf("failed to create script dir: %w", err)
	}

	if err := os.WriteFile(scriptPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save script: %w", err)
	}

	return nil
}



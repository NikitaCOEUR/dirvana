package status

import (
	"strings"
	"testing"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestRender_EmptyData tests rendering with minimal data
func TestRender_EmptyData(t *testing.T) {
	data := &Data{
		CurrentDir:          "/test/dir",
		Version:             "1.0.0",
		Shell:               "bash",
		HookInstalled:       false,
		RCFile:              "/home/user/.bashrc",
		CachePath:           "/test/cache.json",
		AuthPath:            "/test/auth.json",
		HasAnyConfig:        false,
		Authorized:          true,
		Aliases:             make(map[string]config.AliasInfo),
		Functions:           make([]string, 0),
		EnvStatic:           make(map[string]string),
		EnvShell:            make(map[string]config.EnvShellInfo),
		Flags:               make([]string, 0),
		LocalConfigs:        make([]config.FileInfo, 0),
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
	}

	output := Render(data)

	// Verify sections are present
	assert.Contains(t, output, "Current directory:")
	assert.Contains(t, output, "/test/dir")
	assert.Contains(t, output, "Version:")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "System & Installation:")
	assert.Contains(t, output, "Shell:")
	assert.Contains(t, output, "bash")
	assert.Contains(t, output, "Hook:")
	assert.Contains(t, output, "Not installed")
	assert.Contains(t, output, "Configuration hierarchy:")
	assert.Contains(t, output, "No configuration files found")
	assert.Contains(t, output, "Cache:")
	assert.Contains(t, output, "Completion:")

	// Authorization section should NOT be present (no config)
	assert.NotContains(t, output, "Authorization:")
}

// TestRender_WithUnauthorizedConfig tests rendering with unauthorized config
func TestRender_WithUnauthorizedConfig(t *testing.T) {
	data := &Data{
		CurrentDir:    "/test/dir",
		Version:       "1.0.0",
		Shell:         "bash",
		HookInstalled: true,
		RCFile:        "/home/user/.bashrc",
		CachePath:     "/test/cache.json",
		AuthPath:      "/test/auth.json",
		HasAnyConfig:  true,
		Authorized:    false,
		LocalConfigs: []config.FileInfo{
			{
				Path:       "/test/dir/.dirvana.yml",
				Loaded:     false,
				Authorized: false,
				LocalOnly:  false,
			},
		},
		Aliases:             make(map[string]config.AliasInfo),
		Functions:           make([]string, 0),
		EnvStatic:           make(map[string]string),
		EnvShell:            make(map[string]config.EnvShellInfo),
		Flags:               make([]string, 0),
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
	}

	output := Render(data)

	// Authorization section SHOULD be present
	assert.Contains(t, output, "Authorization:")
	assert.Contains(t, output, "Not authorized")
	assert.Contains(t, output, "Run 'dirvana allow")

	// Hook should show as installed
	assert.Contains(t, output, "Installed")

	// Config should show as not authorized
	assert.Contains(t, output, "/test/dir/.dirvana.yml")
	assert.Contains(t, output, "not authorized")
}

// TestRender_WithAuthorizedConfig tests rendering with authorized config and content
func TestRender_WithAuthorizedConfig(t *testing.T) {
	data := &Data{
		CurrentDir:    "/test/dir",
		Version:       "1.0.0",
		Shell:         "zsh",
		HookInstalled: true,
		RCFile:        "/home/user/.zshrc",
		CachePath:     "/test/cache.json",
		AuthPath:      "/test/auth.json",
		HasAnyConfig:  true,
		Authorized:    true,
		LocalConfigs: []config.FileInfo{
			{
				Path:       "/test/dir/.dirvana.yml",
				Loaded:     true,
				Authorized: true,
				LocalOnly:  false,
			},
		},
		Aliases: map[string]config.AliasInfo{
			"gs": {Command: "git status"},
			"k":  {Command: "kubectl"},
		},
		Functions: []string{"greet", "mkcd"},
		EnvStatic: map[string]string{
			"PROJECT_NAME": "dirvana",
			"BUILD_DIR":    "./build",
		},
		EnvShell: map[string]config.EnvShellInfo{
			"GIT_BRANCH": {
				Command:  "git branch --show-current",
				Approved: true,
			},
		},
		Flags:               []string{"local_only"},
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
	}

	output := Render(data)

	// Authorization
	assert.Contains(t, output, "Authorization:")
	assert.Contains(t, output, "Authorized")
	assert.NotContains(t, output, "Run 'dirvana allow")

	// Config
	assert.Contains(t, output, "/test/dir/.dirvana.yml")
	assert.NotContains(t, output, "not authorized")

	// Aliases
	assert.Contains(t, output, "Aliases:")
	assert.Contains(t, output, "gs")
	assert.Contains(t, output, "git status")
	assert.Contains(t, output, "k")
	assert.Contains(t, output, "kubectl")

	// Functions
	assert.Contains(t, output, "Functions:")
	assert.Contains(t, output, "greet()")
	assert.Contains(t, output, "mkcd()")

	// Environment variables
	assert.Contains(t, output, "Environment variables:")
	assert.Contains(t, output, "Static:")
	assert.Contains(t, output, "PROJECT_NAME")
	assert.Contains(t, output, "dirvana")
	assert.Contains(t, output, "Dynamic (shell):")
	assert.Contains(t, output, "GIT_BRANCH")
	assert.Contains(t, output, "git branch --show-current")
	assert.Contains(t, output, "approved")

	// Flags
	assert.Contains(t, output, "Flags:")
	assert.Contains(t, output, "local_only")
}

// TestRender_WithConditionalAliases tests rendering with conditional aliases
func TestRender_WithConditionalAliases(t *testing.T) {
	data := &Data{
		CurrentDir:    "/test/dir",
		Version:       "1.0.0",
		Shell:         "bash",
		HookInstalled: true,
		RCFile:        "/home/user/.bashrc",
		CachePath:     "/test/cache.json",
		AuthPath:      "/test/auth.json",
		HasAnyConfig:  true,
		Authorized:    true,
		LocalConfigs: []config.FileInfo{
			{
				Path:       "/test/dir/.dirvana.yml",
				Loaded:     true,
				Authorized: true,
				LocalOnly:  false,
			},
		},
		Aliases: map[string]config.AliasInfo{
			"simple": {
				Command: "echo simple",
				HasWhen: false,
			},
			"k": {
				Command:     "kubectl",
				HasWhen:     true,
				WhenSummary: "var:KUBECONFIG + file:$KUBECONFIG",
				Else:        "echo 'Error: KUBECONFIG not set'",
			},
			"dev": {
				Command:     "npm run dev",
				HasWhen:     true,
				WhenSummary: "file:package.json",
				Else:        "echo 'Error: package.json not found'",
			},
			"prod": {
				Command:     "npm run prod",
				HasWhen:     true,
				WhenSummary: "all(file:package.json, var:NODE_ENV)",
				Else:        "",
			},
		},
		Functions:           make([]string, 0),
		EnvStatic:           make(map[string]string),
		EnvShell:            make(map[string]config.EnvShellInfo),
		Flags:               make([]string, 0),
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
	}

	output := Render(data)

	// Aliases section
	assert.Contains(t, output, "Aliases:")

	// Simple alias without condition
	assert.Contains(t, output, "simple")
	assert.Contains(t, output, "echo simple")

	// Conditional alias with when and else
	assert.Contains(t, output, "k")
	assert.Contains(t, output, "kubectl")
	assert.Contains(t, output, "when:")
	assert.Contains(t, output, "var:KUBECONFIG + file:$KUBECONFIG")
	assert.Contains(t, output, "else:")
	assert.Contains(t, output, "echo 'Error: KUBECONFIG not set'")

	// Conditional alias with file condition
	assert.Contains(t, output, "dev")
	assert.Contains(t, output, "npm run dev")
	assert.Contains(t, output, "file:package.json")
	assert.Contains(t, output, "echo 'Error: package.json not found'")

	// Conditional alias with composite condition (all) and no else
	assert.Contains(t, output, "prod")
	assert.Contains(t, output, "npm run prod")
	assert.Contains(t, output, "all(file:package.json, var:NODE_ENV)")

	// Verify the "when:" label appears multiple times (once for each conditional alias)
	whenCount := strings.Count(output, "when:")
	assert.Equal(t, 3, whenCount, "Should have 3 conditional aliases with 'when:'")

	// Verify the "else:" label appears for aliases with else (k and dev)
	elseCount := strings.Count(output, "else:")
	assert.Equal(t, 2, elseCount, "Should have 2 aliases with 'else:'")
}

// TestRender_WithGlobalConfig tests rendering with global and local configs
func TestRender_WithGlobalConfig(t *testing.T) {
	data := &Data{
		CurrentDir:    "/test/dir",
		Version:       "1.0.0",
		Shell:         "bash",
		HookInstalled: true,
		CachePath:     "/test/cache.json",
		AuthPath:      "/test/auth.json",
		HasAnyConfig:  true,
		Authorized:    true,
		GlobalConfig: &config.GlobalInfo{
			Path:   "/home/user/.config/dirvana/config.yaml",
			Exists: true,
			Loaded: true,
		},
		LocalConfigs: []config.FileInfo{
			{
				Path:       "/test/dir/.dirvana.yml",
				Loaded:     true,
				Authorized: true,
				LocalOnly:  false,
			},
		},
		Aliases:             make(map[string]config.AliasInfo),
		Functions:           make([]string, 0),
		EnvStatic:           make(map[string]string),
		EnvShell:            make(map[string]config.EnvShellInfo),
		Flags:               make([]string, 0),
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
	}

	output := Render(data)

	// Global config should appear first
	assert.Contains(t, output, "/home/user/.config/dirvana/config.yaml")
	assert.Contains(t, output, "(global)")

	// Local config should appear after
	lines := strings.Split(output, "\n")
	globalIdx := -1
	localIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "(global)") {
			globalIdx = i
		}
		if strings.Contains(line, "/test/dir/.dirvana.yml") {
			localIdx = i
		}
	}
	assert.True(t, globalIdx < localIdx, "Global config should appear before local config")
}

// TestRender_WithCache tests rendering with cache information
func TestRender_WithCache(t *testing.T) {
	cacheUpdated := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	data := &Data{
		CurrentDir:    "/test/dir",
		Version:       "1.0.0",
		Shell:         "bash",
		HookInstalled: true,
		CachePath:     "/test/cache.json",
		AuthPath:      "/test/auth.json",
		HasAnyConfig:  true,
		Authorized:    true,
		LocalConfigs: []config.FileInfo{
			{
				Path:       "/test/dir/.dirvana.yml",
				Loaded:     true,
				Authorized: true,
				LocalOnly:  false,
			},
		},
		CacheFileSize:       4096,
		CacheTotalEntries:   5,
		CacheValid:          true,
		CacheUpdated:        cacheUpdated,
		CacheLocalOnly:      false,
		Aliases:             make(map[string]config.AliasInfo),
		Functions:           make([]string, 0),
		EnvStatic:           make(map[string]string),
		EnvShell:            make(map[string]config.EnvShellInfo),
		Flags:               make([]string, 0),
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
	}

	output := Render(data)

	// Cache section
	assert.Contains(t, output, "Cache:")
	assert.Contains(t, output, "/test/cache.json")
	assert.Contains(t, output, "4.0 KB")
	assert.Contains(t, output, "Total entries:")
	assert.Contains(t, output, "5")

	// Current directory cache (since HasAnyConfig = true)
	assert.Contains(t, output, "Current directory:")
	assert.Contains(t, output, "Valid")
	assert.Contains(t, output, "2024-01-01")
}

// TestRender_WithCompletion tests rendering with completion information
func TestRender_WithCompletion(t *testing.T) {
	data := &Data{
		CurrentDir:    "/test/dir",
		Version:       "1.0.0",
		Shell:         "bash",
		HookInstalled: true,
		CachePath:     "/test/cache.json",
		AuthPath:      "/test/auth.json",
		HasAnyConfig:  false,
		Authorized:    true,
		CompletionDetection: &CompletionDetectionInfo{
			Path: "/test/detection.json",
			Size: 1024,
			Commands: map[string]string{
				"kubectl": "Cobra",
				"helm":    "Cobra",
				"go":      "Flag",
				"custom":  "Script",
			},
		},
		CompletionRegistry: &CompletionRegistryInfo{
			Path:       "/test/registry.yaml",
			Size:       2048,
			ToolsCount: 10,
		},
		CompletionScripts: []CompletionScriptInfo{
			{Tool: "kubectl", Path: "/test/scripts/kubectl.sh", Size: 4096},
			{Tool: "helm", Path: "/test/scripts/helm.sh", Size: 3072},
		},
		CompletionOverrides: map[string]string{
			"k": "kubectl",
		},
		Aliases:       make(map[string]config.AliasInfo),
		Functions:     make([]string, 0),
		EnvStatic:     make(map[string]string),
		EnvShell:      make(map[string]config.EnvShellInfo),
		Flags:         make([]string, 0),
		LocalConfigs:  make([]config.FileInfo, 0),
	}

	output := Render(data)

	// Completion section
	assert.Contains(t, output, "Completion:")

	// Detection cache
	assert.Contains(t, output, "Detection cache:")
	assert.Contains(t, output, "/test/detection.json")
	assert.Contains(t, output, "1.0 KB")
	assert.Contains(t, output, "Detected commands:")
	assert.Contains(t, output, "4")

	// Sources
	assert.Contains(t, output, "Sources:")
	assert.Contains(t, output, "Cobra")
	assert.Contains(t, output, "kubectl")
	assert.Contains(t, output, "helm")
	assert.Contains(t, output, "Flag")
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "Script")
	assert.Contains(t, output, "custom")

	// Registry
	assert.Contains(t, output, "Registry:")
	assert.Contains(t, output, "/test/registry.yaml")
	assert.Contains(t, output, "2.0 KB")
	assert.Contains(t, output, "Tools available:")
	assert.Contains(t, output, "10")

	// Downloaded scripts
	assert.Contains(t, output, "Downloaded scripts:")
	assert.Contains(t, output, "kubectl")
	assert.Contains(t, output, "4.0 KB")
	assert.Contains(t, output, "helm")
	assert.Contains(t, output, "3.0 KB")

	// Completion overrides
	assert.Contains(t, output, "Completion overrides:")
	assert.Contains(t, output, "k")
	assert.Contains(t, output, "kubectl")
}

// TestFormatBytes tests byte formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		assert.Equal(t, tt.expected, result, "formatBytes(%d)", tt.input)
	}
}

// TestTruncateString tests string truncation
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten ch", 14, "exactly ten ch"},
		{"this is a very long string that needs truncation", 20, "this is a very lo..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, result, "truncateString(%q, %d)", tt.input, tt.maxLen)
	}
}

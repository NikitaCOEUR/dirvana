package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
)

// FileInfo represents information about a configuration file
type FileInfo struct {
	Path       string
	Loaded     bool
	Authorized bool
	LocalOnly  bool
}

// GlobalInfo represents information about the global configuration
type GlobalInfo struct {
	Path   string
	Exists bool
	Loaded bool
}

// HierarchyInfo contains information about the configuration hierarchy
type HierarchyInfo struct {
	GlobalConfig *GlobalInfo
	LocalConfigs []FileInfo
	MergedConfig *Config
}

// GetHierarchyInfo returns information about the configuration hierarchy for a directory
func GetHierarchyInfo(currentDir string, authMgr *auth.Auth) (*HierarchyInfo, error) {
	loader := New()

	// Load config hierarchy with auth
	merged, loadedConfigFiles, err := loader.LoadHierarchyWithAuth(currentDir, authMgr)
	if err != nil {
		return nil, err
	}

	info := &HierarchyInfo{
		LocalConfigs: make([]FileInfo, 0),
		MergedConfig: merged,
	}

	// Find all config files in the hierarchy
	allConfigFiles, _ := FindConfigFiles(currentDir)

	// Check global config
	globalPath, err := GetGlobalConfigPath()
	if err == nil {
		if _, err := os.Stat(globalPath); err == nil {
			globalLoaded := false
			// Check if global was actually loaded
			for _, loaded := range loadedConfigFiles {
				if loaded == globalPath {
					globalLoaded = true
					break
				}
			}

			info.GlobalConfig = &GlobalInfo{
				Path:   globalPath,
				Exists: true,
				Loaded: globalLoaded,
			}
		}
	}

	// Process local config files
	for _, path := range allConfigFiles {
		configDir := filepath.Dir(path)

		// Check if this config was actually loaded
		loaded := false
		for _, loadedPath := range loadedConfigFiles {
			if loadedPath == path {
				loaded = true
				break
			}
		}

		// Check if this directory is authorized
		authorized, _ := authMgr.IsAllowed(configDir)

		localOnly := false
		if merged != nil && merged.LocalOnly && path == allConfigFiles[len(allConfigFiles)-1] {
			localOnly = true
		}

		info.LocalConfigs = append(info.LocalConfigs, FileInfo{
			Path:       path,
			Loaded:     loaded,
			Authorized: authorized,
			LocalOnly:  localOnly,
		})
	}

	return info, nil
}

// AliasInfo contains information about a single alias
type AliasInfo struct {
	Command    string
	HasWhen    bool
	WhenSummary string
	Else       string
}

// DetailsInfo contains detailed information about the merged configuration
type DetailsInfo struct {
	Aliases   map[string]AliasInfo
	Functions []string
	EnvStatic map[string]string
	EnvShell  map[string]EnvShellInfo
	Flags     []string
}

// EnvShellInfo represents information about a shell environment variable
type EnvShellInfo struct {
	Command  string
	Approved bool
}

// GetConfigDetails extracts detailed information from a merged configuration
func GetConfigDetails(merged *Config, authMgr *auth.Auth, currentDir string) *DetailsInfo {
	if merged == nil {
		return &DetailsInfo{
			Aliases:   make(map[string]AliasInfo),
			Functions: make([]string, 0),
			EnvStatic: make(map[string]string),
			EnvShell:  make(map[string]EnvShellInfo),
			Flags:     make([]string, 0),
		}
	}

	details := &DetailsInfo{
		Aliases:   convertAliasesWithInfo(merged.GetAliases()),
		Functions: getFunctionsList(merged.Functions),
		EnvShell:  make(map[string]EnvShellInfo),
		Flags:     make([]string, 0),
	}

	// Get environment variables
	staticEnv, shellEnv := merged.GetEnvVars()
	details.EnvStatic = staticEnv

	// Get shell env vars with approval status
	var shellApproved bool
	if authMgr != nil {
		dirAuth := authMgr.GetAuth(currentDir)
		shellApproved = dirAuth != nil && dirAuth.ShellCommandsHash != ""
	}

	for name, cmd := range shellEnv {
		details.EnvShell[name] = EnvShellInfo{
			Command:  cmd,
			Approved: shellApproved,
		}
	}

	// Get flags
	if merged.LocalOnly {
		details.Flags = append(details.Flags, "local_only")
	}
	if merged.IgnoreGlobal {
		details.Flags = append(details.Flags, "ignore_global")
	}

	return details
}

// GetCompletionOverrides extracts completion overrides from aliases
func GetCompletionOverrides(merged *Config) map[string]string {
	if merged == nil {
		return make(map[string]string)
	}

	result := make(map[string]string)
	for aliasName, aliasValue := range merged.Aliases {
		switch v := aliasValue.(type) {
		case map[string]interface{}:
			if cc, ok := v["completion"].(string); ok {
				result[aliasName] = cc
			}
		}
	}
	return result
}

// Helper to convert aliases with full information including conditions
func convertAliasesWithInfo(aliases map[string]AliasConfig) map[string]AliasInfo {
	result := make(map[string]AliasInfo)
	for name, aliasConfig := range aliases {
		info := AliasInfo{
			Command: aliasConfig.Command,
			HasWhen: aliasConfig.When != nil,
			Else:    aliasConfig.Else,
		}

		// Generate when summary
		if aliasConfig.When != nil {
			info.WhenSummary = summarizeWhen(aliasConfig.When)
		}

		result[name] = info
	}
	return result
}

// Helper to convert aliases (legacy, kept for compatibility)
func convertAliases(aliases map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for name, value := range aliases {
		var cmd string
		switch v := value.(type) {
		case string:
			cmd = v
		case map[string]interface{}:
			if c, ok := v["command"].(string); ok {
				cmd = c
			}
		}
		if cmd != "" {
			result[name] = cmd
		}
	}
	return result
}

// summarizeWhen creates a human-readable summary of a When condition
func summarizeWhen(when *When) string {
	if when == nil {
		return ""
	}

	var parts []string

	// Atomic conditions
	if when.File != "" {
		parts = append(parts, fmt.Sprintf("file:%s", when.File))
	}
	if when.Var != "" {
		parts = append(parts, fmt.Sprintf("var:%s", when.Var))
	}
	if when.Dir != "" {
		parts = append(parts, fmt.Sprintf("dir:%s", when.Dir))
	}
	if when.Command != "" {
		parts = append(parts, fmt.Sprintf("cmd:%s", when.Command))
	}

	// Composite conditions
	if len(when.All) > 0 {
		subParts := make([]string, len(when.All))
		for i, sub := range when.All {
			subParts[i] = summarizeWhen(&sub)
		}
		parts = append(parts, fmt.Sprintf("all(%s)", strings.Join(subParts, ", ")))
	}
	if len(when.Any) > 0 {
		subParts := make([]string, len(when.Any))
		for i, sub := range when.Any {
			subParts[i] = summarizeWhen(&sub)
		}
		parts = append(parts, fmt.Sprintf("any(%s)", strings.Join(subParts, " | ")))
	}

	if len(parts) == 0 {
		return ""
	}

	// If multiple atomic conditions, they're ANDed together
	if len(parts) > 1 && len(when.All) == 0 && len(when.Any) == 0 {
		return strings.Join(parts, " + ")
	}

	return strings.Join(parts, ", ")
}

// Helper to get functions list
func getFunctionsList(functions map[string]string) []string {
	result := make([]string, 0, len(functions))
	for name := range functions {
		result = append(result, name)
	}
	return result
}

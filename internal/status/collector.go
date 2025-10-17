// Package status provides status information collection and display for Dirvana.
package status

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/completion"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
)

const (
	shellBash = "bash"
	shellZsh  = "zsh"
)

// CollectAll gathers all status information from the current directory
func CollectAll(cachePath, authPath string) (*Data, error) {
	data := &Data{
		Aliases:             make(map[string]string),
		Functions:           make([]string, 0),
		EnvStatic:           make(map[string]string),
		EnvShell:            make(map[string]config.EnvShellInfo),
		Flags:               make([]string, 0),
		LocalConfigs:        make([]config.FileInfo, 0),
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
		CachePath:           cachePath,
		AuthPath:            authPath,
		Version:             version.Version,
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	data.CurrentDir = currentDir

	// Collect system info
	collectSystemInfo(data)

	// Initialize components
	cacheObj, err := cache.New(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	authMgr, err := auth.New(authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	// Collect config hierarchy info from config module
	hierarchyInfo, err := config.GetHierarchyInfo(currentDir, authMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to get config hierarchy: %w", err)
	}

	// Set global config info
	data.GlobalConfig = hierarchyInfo.GlobalConfig

	// Set local configs info
	data.LocalConfigs = hierarchyInfo.LocalConfigs

	// Check authorization only if there are configs
	hasConfigs := len(data.LocalConfigs) > 0 || (data.GlobalConfig != nil && data.GlobalConfig.Exists)
	data.HasAnyConfig = hasConfigs

	if hasConfigs {
		allowed, err := authMgr.IsAllowed(currentDir)
		if err != nil {
			return nil, fmt.Errorf("failed to check authorization: %w", err)
		}
		data.Authorized = allowed

		// If authorized and configs exist, collect details
		if allowed {
			// Get config details from config module
			details := config.GetConfigDetails(hierarchyInfo.MergedConfig, authMgr, currentDir)
			data.Aliases = details.Aliases
			data.Functions = details.Functions
			data.EnvStatic = details.EnvStatic
			data.EnvShell = details.EnvShell
			data.Flags = details.Flags

			// Get completion overrides
			data.CompletionOverrides = config.GetCompletionOverrides(hierarchyInfo.MergedConfig)
		}
	} else {
		// No configs, authorization is not applicable
		data.Authorized = true
	}

	// Always collect cache info
	collectCacheInfo(data, cacheObj, currentDir)

	// Collect completion info
	collectCompletionInfo(data)

	return data, nil
}

func collectSystemInfo(data *Data) {
	// Detect current shell
	shell := os.Getenv("SHELL")
	shellName := "unknown"
	if strings.Contains(shell, shellBash) {
		shellName = shellBash
	} else if strings.Contains(shell, shellZsh) {
		shellName = shellZsh
	}
	data.Shell = shellName

	// Check if hook is installed
	hookInstalled := false
	rcFile := ""
	if shellName == shellBash || shellName == shellZsh {
		home, err := os.UserHomeDir()
		if err == nil {
			switch shellName {
			case shellBash:
				rcFile = filepath.Join(home, ".bashrc")
			case shellZsh:
				rcFile = filepath.Join(home, ".zshrc")
			}

			if rcFile != "" {
				hookInstalled = checkRCFileForHook(rcFile)
			}
		}
	}

	data.HookInstalled = hookInstalled
	data.RCFile = rcFile
}

// checkRCFileForHook scans RC file line by line for hook patterns (optimized)
func checkRCFileForHook(rcFile string) bool {
	file, err := os.Open(rcFile)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	// Hook patterns to search for
	patterns := []string{
		"# Dirvana",
		"hook-bash.sh",
		"hook-zsh.sh",
		"dirvana export",
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check each pattern
		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				return true
			}
		}
	}

	return false
}

func collectCacheInfo(data *Data, cacheObj *cache.Cache, currentDir string) {
	// Get cache file info from cache module
	cacheInfo, err := cache.GetCacheInfo(data.CachePath)
	if err == nil && cacheInfo != nil {
		data.CacheFileSize = cacheInfo.Size
		data.CacheTotalEntries = cacheInfo.TotalEntries
	}

	// Check current directory cache status
	if len(data.LocalConfigs) > 0 {
		configHash := ""
		mainConfig := data.LocalConfigs[len(data.LocalConfigs)-1].Path
		cfgLoader := config.New()
		configHash, _ = cfgLoader.Hash(mainConfig)

		cacheValid := cacheObj.IsValid(currentDir, configHash, version.Version)
		data.CacheValid = cacheValid

		if cacheValid {
			if entry, ok := cacheObj.Get(currentDir); ok {
				data.CacheUpdated = entry.Timestamp
				data.CacheLocalOnly = entry.LocalOnly
			}
		}
	}
}

func collectCompletionInfo(data *Data) {
	cacheDir := filepath.Dir(data.CachePath)

	// Get detection cache info from completion module
	if detectionInfo, err := completion.GetDetectionCacheInfo(cacheDir); err == nil && detectionInfo != nil {
		data.CompletionDetection = &CompletionDetectionInfo{
			Path:     detectionInfo.Path,
			Size:     detectionInfo.Size,
			Commands: detectionInfo.Commands,
		}
	}

	// Get registry info from completion module
	if registryInfo, err := completion.GetRegistryInfo(cacheDir); err == nil && registryInfo != nil {
		data.CompletionRegistry = &CompletionRegistryInfo{
			Path:       registryInfo.Path,
			Size:       registryInfo.Size,
			ToolsCount: registryInfo.ToolsCount,
		}
	}

	// Get downloaded scripts from completion module
	if scripts, err := completion.GetDownloadedScripts(cacheDir); err == nil && scripts != nil {
		for _, script := range scripts {
			data.CompletionScripts = append(data.CompletionScripts, CompletionScriptInfo{
				Tool: script.Tool,
				Path: script.Path,
				Size: script.Size,
			})
		}
	}
}

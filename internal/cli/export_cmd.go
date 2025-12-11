package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/derrors"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
	"github.com/NikitaCOEUR/dirvana/internal/shellctx"
	"github.com/NikitaCOEUR/dirvana/internal/timing"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
)

// ExportParams contains parameters for the Export command
type ExportParams struct {
	LogLevel  string
	PrevDir   string
	CachePath string
	AuthPath  string
}

// activeChains holds previous and current active config chains
type activeChains struct {
	prev    []string
	current []string
}

// calculateActiveChains computes active config chains for previous and current directories
func calculateActiveChains(prevDir, currentDir string, authMgr *auth.Auth, configLoader *config.Loader) activeChains {
	chains := activeChains{}

	if prevDir != "" && prevDir != currentDir {
		chains.prev = shellctx.GetActiveConfigChain(prevDir, authMgr, configLoader)
		chains.current = shellctx.GetActiveConfigChain(currentDir, authMgr, configLoader)
	} else {
		// Same directory or no previous directory
		chains.current = shellctx.GetActiveConfigChain(currentDir, authMgr, configLoader)
	}

	return chains
}

// generateCleanupCodeForDirs generates cleanup code for directories that need cleanup
func generateCleanupCodeForDirs(cleanupDirs []string, cacheStorage *cache.Cache, shell string, log *logger.Logger) string {
	var cleanupCode string

	if len(cleanupDirs) == 0 {
		return cleanupCode
	}

	// Cleanup each directory individually
	for _, dir := range cleanupDirs {
		if entry, found := cacheStorage.Get(dir); found {
			startTime := time.Now()
			cleanupCode += shellctx.GenerateCleanupCode(
				entry.Aliases,
				entry.Functions,
				entry.EnvVars,
				shell,
			)
			duration := time.Since(startTime)

			log.Debug().
				Str("cleanup_dir", dir).
				Dur("cleanup_ms", duration).
				Int("aliases", len(entry.Aliases)).
				Int("functions", len(entry.Functions)).
				Int("env_vars", len(entry.EnvVars)).
				Msg("Cleaning up config")
		}
	}

	return cleanupCode
}

// detectTargetShell determines the target shell for code generation
func detectTargetShell() string {
	targetShell := DetectShell("auto")

	if targetShell == ShellBash {
		// Check if it's a real detection or just the default fallback
		if os.Getenv("DIRVANA_SHELL") == "" &&
			detectShellFromParentProcess() == "" &&
			!containsString(os.Getenv("SHELL"), "bash") {
			targetShell = "" // Generate for all shells
		}
	}

	return targetShell
}

// checkUnauthorizedConfig warns if current directory has an unauthorized config
func checkUnauthorizedConfig(currentDir string, currentActiveChain []string, targetShell string, log *logger.Logger) {
	if !config.HasLocalConfig(currentDir) {
		return
	}

	// Check if currentDir is in the active chain
	isInActiveChain := false
	for _, dir := range currentActiveChain {
		if dir == currentDir {
			isInActiveChain = true
			break
		}
	}

	if !isInActiveChain {
		// Current directory has local config but is not authorized
		suggestion := "dirvana allow " + currentDir
		if targetShell != "" {
			suggestion += "\n💡 Then reload with: eval \"$(DIRVANA_SHELL=" + targetShell + " dirvana export)\""
		} else {
			suggestion += "\n💡 Then reload with: eval \"$(dirvana export)\""
		}

		log.Warn().
			Str("dir", currentDir).
			Msg("Local dirvana config found but directory not authorized. Run: " + suggestion)
	}
}

// loadAndMergeConfigs loads all configs in the active chain and caches them
// Uses LoadHierarchyWithAuth to properly handle global config, ignore_global, and local_only
func loadAndMergeConfigs(currentActiveChain []string, comps *components, log *logger.Logger, currentDir string) *config.Config {
	// Load the full hierarchy with proper global, ignore_global, and local_only handling
	mergedConfig, _, err := comps.config.LoadHierarchyWithAuth(currentDir, comps.auth)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load config hierarchy")
		return nil
	}

	// Cache individual configs for cleanup purposes
	// We iterate through the active chain to cache each config separately
	for _, configDir := range currentActiveChain {
		// Find config file in this directory
		var configPath string
		for _, name := range config.SupportedConfigNames {
			path := filepath.Join(configDir, name)
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			continue
		}

		// Load this specific config (not hierarchy) for caching
		cfg, err := comps.config.Load(configPath)
		if err != nil {
			log.Warn().Err(err).Str("path", configPath).Msg("Failed to load config")
			continue
		}

		// Cache individual config definitions for future cleanup
		hash, _ := comps.config.Hash(configPath)
		staticEnv, shellEnv := cfg.GetEnvVars()
		aliases := cfg.GetAliases()
		aliasKeys := keysFromAliasMap(aliases)
		functions := keysFromMap(cfg.Functions)
		envVars := mergeTwoKeyLists(staticEnv, shellEnv)
		commandMap := buildCommandMap(aliases, cfg.Functions)
		completionMap := buildCompletionMap(aliases)

		entry := &cache.Entry{
			Path:          configDir,
			Hash:          hash,
			Timestamp:     time.Now(),
			Version:       version.Version,
			LocalOnly:     cfg.LocalOnly,
			Aliases:       aliasKeys,
			Functions:     functions,
			EnvVars:       envVars,
			CommandMap:    commandMap,
			CompletionMap: completionMap,
			// ShellCode is not stored for individual configs
		}

		if err := comps.cache.Set(entry); err != nil {
			log.Warn().Err(err).Str("dir", configDir).Msg("Failed to update cache")
		}
	}

	return mergedConfig
}

// cacheMergedConfig creates and caches the merged configuration for the current directory
func cacheMergedConfig(currentDir string, hierarchyHash string, hierarchyPaths []string, mergedConfig *config.Config, aliases map[string]config.AliasConfig, mergedCommandMap, mergedCompletionMap map[string]string, comps *components, log *logger.Logger) {
	if hierarchyHash == "" {
		return
	}

	// Check if current directory has a local config file
	hasLocalConfig := config.HasLocalConfig(currentDir)

	// Only extract cleanup data if this directory has a local config
	// Directories without local config still get cached for performance (completion/exec),
	// but without cleanup data since they only inherit configs (nothing new to clean up)
	var aliasKeys, functions, envVars []string
	if hasLocalConfig {
		aliasKeys = keysFromAliasMap(aliases)
		functions = keysFromMap(mergedConfig.Functions)
		staticEnv, shellEnv := mergedConfig.GetEnvVars()
		envVars = mergeTwoKeyLists(staticEnv, shellEnv)
	}

	mergedEntry := &cache.Entry{
		Path:                currentDir,
		Hash:                hierarchyHash,
		Timestamp:           time.Now(),
		Version:             version.Version,
		MergedCommandMap:    mergedCommandMap,
		MergedCompletionMap: mergedCompletionMap,
		HierarchyHash:       hierarchyHash,
		HierarchyPaths:      hierarchyPaths,
		// Store cleanup data only for directories with local config
		// This avoids duplicating cleanup data for inherited configs
		Aliases:   aliasKeys, // nil if !hasLocalConfig
		Functions: functions, // nil if !hasLocalConfig
		EnvVars:   envVars,   // nil if !hasLocalConfig
	}

	if err := comps.cache.Set(mergedEntry); err != nil {
		log.Debug().Err(err).Str("dir", currentDir).Msg("Failed to cache merged config")
	} else {
		logEvent := log.Debug().
			Str("dir", currentDir).
			Bool("has_local_config", hasLocalConfig).
			Int("merged_commands", len(mergedCommandMap)).
			Int("merged_completions", len(mergedCompletionMap)).
			Str("hierarchy_hash", hierarchyHash)

		if hasLocalConfig {
			logEvent.
				Int("aliases", len(aliasKeys)).
				Int("functions", len(functions)).
				Int("env_vars", len(envVars))
		}

		logEvent.Msg("Cached merged configuration")
	}
}

// Export generates and outputs shell code for the current directory
func Export(params ExportParams) error {
	// Check if Dirvana is disabled via environment variable
	if os.Getenv("DIRVANA_ENABLED") == "false" {
		// Return empty output (no error) so shell hook succeeds silently
		fmt.Print("")
		return nil
	}

	timer := timing.NewTimer()
	log := logger.New(params.LogLevel, os.Stderr)

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return derrors.NewExecutionError("export", "failed to get current directory", err)
	}

	log.Debug().Str("dir", currentDir).Str("prev", params.PrevDir).Msg("Exporting shell code")

	// Initialize components
	comps, err := initializeComponents(params.CachePath, params.AuthPath)
	if err != nil {
		return err
	}
	timer.Mark("init")

	// Detect current shell early (for cleanup and shell-specific code generation)
	targetShell := detectTargetShell()

	// Calculate active config chains for cleanup logic
	chains := calculateActiveChains(params.PrevDir, currentDir, comps.auth, comps.config)
	timer.Mark("calc_chains")

	// Determine what needs cleanup
	cleanupDirs := shellctx.CalculateCleanup(chains.prev, chains.current)
	cleanupCode := generateCleanupCodeForDirs(cleanupDirs, comps.cache, targetShell, log)
	timer.Mark("cleanup")

	// If no active configs in current directory, just output cleanup and return
	if len(chains.current) == 0 {
		if cleanupCode != "" {
			fmt.Print(cleanupCode)
		} else {
			fmt.Print("") // Output empty string so shell hook doesn't fail
		}
		return nil
	}

	// Check if current directory has a local config but is not in the active chain
	checkUnauthorizedConfig(currentDir, chains.current, targetShell, log)

	// Load each config in the active chain and cache individual definitions
	// This now uses LoadHierarchyWithAuth to properly handle global config, ignore_global, and local_only
	mergedConfig := loadAndMergeConfigs(chains.current, comps, log, currentDir)

	// If no valid configs loaded, output cleanup and return
	if mergedConfig == nil {
		if cleanupCode != "" {
			fmt.Print(cleanupCode)
		} else {
			fmt.Print("")
		}
		return nil
	}
	timer.Mark("load_configs")

	// Cache the merged configuration for fast completion/exec access
	// Build merged command and completion maps from the final merged config
	aliases := mergedConfig.GetAliases()
	mergedCommandMap := buildCommandMap(aliases, mergedConfig.Functions)
	mergedCompletionMap := buildCompletionMap(aliases)

	// Compute hierarchy hash from all active config paths
	hierarchyHash, hierarchyPaths, err := computeHierarchyHash(chains.current, comps.config)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to compute hierarchy hash")
	}

	// Cache the merged result for the current directory
	cacheMergedConfig(currentDir, hierarchyHash, hierarchyPaths, mergedConfig, aliases, mergedCommandMap, mergedCompletionMap, comps, log)
	timer.Mark("cache_merged")

	// Get environment variables and aliases
	staticEnv, shellEnv := mergedConfig.GetEnvVars()

	// Shell command approval logic
	if comps.auth.RequiresShellApproval(currentDir, shellEnv) {
		// Show shell commands for approval
		if err := displayShellCommandsForApproval(shellEnv); err != nil {
			return err
		}
		// Prompt user for approval
		approved, err := promptShellApproval()
		if err != nil {
			return err
		}
		if !approved {
			return derrors.NewShellApprovalError(currentDir, "shell commands not approved", nil)
		}
		// Save approval
		if err := comps.auth.ApproveShellCommands(currentDir, shellEnv); err != nil {
			return err
		}

		// Display confirmation message directly to terminal
		tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil {
			// Fallback to stderr if /dev/tty is not available
			_, _ = fmt.Fprintf(os.Stderr, "\n✓ Shell commands approved and cached\n\n")
		} else {
			_, _ = fmt.Fprintf(tty, "\n✓ Shell commands approved and cached\n\n")
			_ = tty.Close()
		}
	}

	// Configure shell generator
	if targetShell != "" {
		comps.shell.WithShell(targetShell)
	}

	// Generate shell code from merged config
	shellCode := comps.shell.Generate(aliases, mergedConfig.Functions, staticEnv, shellEnv)
	timer.Mark("generate_shell")

	// Prepend cleanup code if needed
	if cleanupCode != "" {
		shellCode = cleanupCode + "\n" + shellCode
	}

	// Log timing information in debug mode
	totalDur := timer.Elapsed()
	log.Debug().
		Int("active_configs", len(chains.current)).
		Int("cleanup_configs", len(cleanupDirs)).
		Int("aliases", len(aliases)).
		Int("functions", len(mergedConfig.Functions)).
		Int("static_env", len(staticEnv)).
		Int("shell_env", len(shellEnv)).
		Dur("total_ms", totalDur).
		Str("timing", timer.Summary()).
		Msg("Generated shell code")

	fmt.Print(shellCode)
	return nil
}

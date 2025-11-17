package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	dircontext "github.com/NikitaCOEUR/dirvana/internal/context"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
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
		chains.prev = dircontext.GetActiveConfigChain(prevDir, authMgr, configLoader)
		chains.current = dircontext.GetActiveConfigChain(currentDir, authMgr, configLoader)
	} else {
		// Same directory or no previous directory
		chains.current = dircontext.GetActiveConfigChain(currentDir, authMgr, configLoader)
	}

	return chains
}

// generateCleanupCodeForDirs generates cleanup code for directories that need cleanup
func generateCleanupCodeForDirs(cleanupDirs []string, cacheStorage *cache.Cache, log *logger.Logger) string {
	var cleanupCode string

	if len(cleanupDirs) == 0 {
		return cleanupCode
	}

	// Cleanup each directory individually
	for _, dir := range cleanupDirs {
		if entry, found := cacheStorage.Get(dir); found {
			startTime := time.Now()
			cleanupCode += dircontext.GenerateCleanupCode(
				entry.Aliases,
				entry.Functions,
				entry.EnvVars,
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

	if targetShell == "bash" {
		// Check if it's a real detection or just the default fallback
		if os.Getenv("DIRVANA_SHELL") == "" &&
			detectShellFromParentProcess() == "" &&
			!strings.Contains(os.Getenv("SHELL"), "bash") {
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
		Aliases:   aliasKeys,   // nil if !hasLocalConfig
		Functions: functions,    // nil if !hasLocalConfig
		EnvVars:   envVars,      // nil if !hasLocalConfig
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
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	log.Debug().Str("dir", currentDir).Str("prev", params.PrevDir).Msg("Exporting shell code")

	// Initialize components
	comps, err := initializeComponents(params.CachePath, params.AuthPath)
	if err != nil {
		return err
	}
	timer.Mark("init")

	// Calculate active config chains for cleanup logic
	chains := calculateActiveChains(params.PrevDir, currentDir, comps.auth, comps.config)
	timer.Mark("calc_chains")

	// Determine what needs cleanup
	cleanupDirs := dircontext.CalculateCleanup(chains.prev, chains.current)
	cleanupCode := generateCleanupCodeForDirs(cleanupDirs, comps.cache, log)
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

	// Detect current shell early (for error messages and shell-specific code generation)
	targetShell := detectTargetShell()

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
			return fmt.Errorf("shell commands not approved")
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

// Display dynamic shell commands for approval
func displayShellCommandsForApproval(shellEnv map[string]string) error {
	if len(shellEnv) == 0 {
		return nil
	}

	// Open /dev/tty to write directly to the terminal
	// This ensures messages are visible even when stdout/stderr are redirected (e.g., in eval)
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		// Fallback to stderr if /dev/tty is not available
		tty = os.Stderr
	} else {
		defer func() { _ = tty.Close() }()
	}

	_, _ = fmt.Fprintf(tty, "\n⚠️  This configuration contains dynamic shell commands:\n\n")
	keys := make([]string, 0, len(shellEnv))
	for k := range shellEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _ = fmt.Fprintf(tty, "   • %s: %s\n", key, shellEnv[key])
	}
	_, _ = fmt.Fprintf(tty, "\nThese commands will execute to set environment variables.\n")
	return nil
}

// Prompt user for shell command approval
func promptShellApproval() (bool, error) {
	// For testing: use stdin/stderr fallback if DIRVANA_TEST_MODE is set
	useFallback := os.Getenv("DIRVANA_TEST_MODE") != ""

	// Open /dev/tty for both reading and writing to interact with the user
	// This ensures prompts are visible even when stdout/stderr are redirected (e.g., in eval)
	var tty *os.File
	var err error

	if !useFallback {
		tty, err = os.OpenFile("/dev/tty", os.O_RDWR, 0)
	} else {
		err = fmt.Errorf("test mode: skip /dev/tty")
	}

	if err != nil {
		// Fallback to stderr for output and stdin for input
		_, _ = fmt.Fprintf(os.Stderr, "Approve execution? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		response = strings.TrimSpace(strings.ToLower(response))
		return response == "y" || response == "yes", nil
	}
	defer func() { _ = tty.Close() }()

	_, _ = fmt.Fprintf(tty, "Approve execution? [y/N]: ")
	reader := bufio.NewReader(tty)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// AllowParams contains parameters for the Allow command
type AllowParams struct {
	AuthPath         string
	PathToAllow      string
	CachePath        string
	LogLevel         string
	AutoApproveShell bool
}

// Allow authorizes a directory for Dirvana execution
func Allow(authPath, pathToAllow string) error {
	return AllowWithParams(AllowParams{
		AuthPath:    authPath,
		PathToAllow: pathToAllow,
		LogLevel:    "warn",
	})
}

// AllowWithParams authorizes a directory and optionally loads the environment
func AllowWithParams(params AllowParams) error {
	authMgr, err := auth.New(params.AuthPath)
	if err != nil {
		return fmt.Errorf("failed to initialize auth: %w", err)
	}

	if err := authMgr.Allow(params.PathToAllow); err != nil {
		return fmt.Errorf("failed to authorize: %w", err)
	}

	// Invalidate cache for the authorized directory
	// This ensures the config will be reloaded with proper authorization
	if params.CachePath != "" {
		cacheStorage, err := cache.New(params.CachePath)
		if err == nil {
			if err := cacheStorage.Delete(params.PathToAllow); err != nil {
				// Log but don't fail - cache invalidation is not critical
				fmt.Fprintf(os.Stderr, "Warning: failed to invalidate cache: %v\n", err)
			}
		}
	}

	fmt.Printf("Authorized: %s\n", params.PathToAllow)

	// If auto-approve flag is set, approve shell commands immediately
	if params.AutoApproveShell {
		if err := approveShellCommandsForPath(params.PathToAllow, authMgr, params.LogLevel); err != nil {
			return fmt.Errorf("failed to auto-approve shell commands: %w", err)
		}
		fmt.Println("✓ Shell commands auto-approved")
	}

	// If we're in the authorized directory, suggest loading the environment
	currentDir, err := os.Getwd()
	if err == nil && currentDir == params.PathToAllow {
		fmt.Println("\n💡 Tip: Run 'eval \"$(dirvana export)\"' to load the environment in your current shell")
		fmt.Println("\tOr run 'cd ..' then 'cd -' to reload the environment")
	}

	return nil
}

// RevokeParams contains parameters for the Revoke command
type RevokeParams struct {
	AuthPath     string
	PathToRevoke string
	CachePath    string
	LogLevel     string
}

// Revoke removes authorization for a directory
func Revoke(authPath, pathToRevoke string) error {
	return RevokeWithParams(RevokeParams{
		AuthPath:     authPath,
		PathToRevoke: pathToRevoke,
		LogLevel:     "warn",
	})
}

// RevokeWithParams removes authorization and optionally unloads the environment
func RevokeWithParams(params RevokeParams) error {
	currentDir, _ := os.Getwd()

	authMgr, err := auth.New(params.AuthPath)
	if err != nil {
		return fmt.Errorf("failed to initialize auth: %w", err)
	}

	if err := authMgr.Revoke(params.PathToRevoke); err != nil {
		return fmt.Errorf("failed to revoke: %w", err)
	}

	// Invalidate cache for the revoked directory and all its subdirectories
	// This ensures configs are no longer accessible without re-authorization
	if params.CachePath != "" {
		cacheStorage, err := cache.New(params.CachePath)
		if err == nil {
			if err := cacheStorage.DeleteWithSubdirs(params.PathToRevoke); err != nil {
				// Log but don't fail - cache invalidation is not critical
				fmt.Fprintf(os.Stderr, "Warning: failed to invalidate cache: %v\n", err)
			}
		}
	}

	fmt.Printf("Revoked: %s\n", params.PathToRevoke)

	// Show cleanup tip if we're in the revoked directory
	if currentDir == params.PathToRevoke {
		fmt.Println("\n💡 Tip: Run 'cd ..' then 'cd -' to unload the Dirvana environment")
		fmt.Println("   Or run: 'eval \"$(dirvana export)\"' to reload the environment if you have parent configs")
	}

	return nil
}

// approveShellCommandsForPath is a helper that loads config and approves shell commands
func approveShellCommandsForPath(path string, authMgr *auth.Auth, logLevel string) error {
	log := logger.New(logLevel, os.Stderr)

	// Initialize config loader
	configLoader := config.New()

	// Load config for this directory
	cfg, err := configLoader.Load(filepath.Join(path, ".dirvana.yml"))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get shell environment variables
	_, shellEnv := cfg.GetEnvVars()

	// If no shell commands, nothing to approve
	if len(shellEnv) == 0 {
		log.Debug().Msg("No shell commands found in config")
		return nil
	}

	// Approve the shell commands
	if err := authMgr.ApproveShellCommands(path, shellEnv); err != nil {
		return fmt.Errorf("failed to approve shell commands: %w", err)
	}

	return nil
}

// List displays all authorized directories
func List(authPath string) error {
	authMgr, err := auth.New(authPath)
	if err != nil {
		return fmt.Errorf("failed to initialize auth: %w", err)
	}

	paths := authMgr.List()
	if len(paths) == 0 {
		fmt.Println("No authorized projects")
		return nil
	}

	fmt.Println("Authorized projects:")
	for _, path := range paths {
		fmt.Printf("  %s\n", path)
	}

	return nil
}

// Init creates a sample .dirvana.yml config file in the current directory or global config
func Init(global bool) error {
	var configPath string

	if global {
		// Create global config
		globalPath, err := config.GetGlobalConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get global config path: %w", err)
		}
		configPath = globalPath

		// Create directory if it doesn't exist
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	} else {
		// Create local config
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		configPath = filepath.Join(currentDir, ".dirvana.yml")
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists: %s", configPath)
	}

	sampleConfig := `# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json
# Dirvana configuration file
# Documentation: https://github.com/NikitaCOEUR/dirvana

# Shell aliases
aliases:
  # Simple string aliases (auto-detects completion)
  # g: git

  # Advanced format with completion control
  # tf:
  #  command: task terraform --
  #  completion: terraform  # Inherits terraform's auto-completion

# Shell functions - reusable command sequences with parameters
functions:
  # Simple greeting function
  # greet: |
  #   echo "Hello, $1!"

# Environment variables
env:
  # Static values
  # PROJECT_NAME: myproject

  # Dynamic values from shell commands (evaluated on load)
  # CURRENT_USER:
  #	  sh: whoami

# Configuration flags
# Set to true to ignore parent configs (only use this directory's config)
# local_only: false

# Set to true to ignore global config (~/.config/dirvana/global.yml)
# ignore_global: false
`

	if err := os.WriteFile(configPath, []byte(sampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	if global {
		fmt.Printf("Created global config: %s\n", configPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit the config file to suit your needs")
		fmt.Println("  2. Run 'dirvana edit --global' to edit the global config")
		fmt.Println("  3. The global config will be automatically loaded in all directories")
	} else {
		fmt.Printf("Created sample config: %s\n", configPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit the config file to suit your needs")
		fmt.Println("  2. Run 'dirvana allow' to authorize this directory")
		fmt.Println("  3. Run 'dirvana setup' to install the shell hook")
	}

	return nil
}

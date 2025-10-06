package cli

import (
	"fmt"
	"os"
	"path/filepath"
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
			cleanupCode += dircontext.GenerateCleanupCode(
				entry.Aliases,
				entry.Functions,
				entry.EnvVars,
			)
			log.Debug().Str("cleanup_dir", dir).Msg("Cleaning up config")
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
func loadAndMergeConfigs(currentActiveChain []string, comps *components, log *logger.Logger) *config.Config {
	var mergedConfig *config.Config

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

		// Load this specific config (not hierarchy)
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

		// Merge configs
		if mergedConfig == nil {
			mergedConfig = cfg
		} else {
			mergedConfig = config.Merge(mergedConfig, cfg)
		}

		// If this config has local_only, stop merging (shouldn't happen as GetActiveConfigChain handles it)
		if cfg.LocalOnly {
			break
		}
	}

	return mergedConfig
}

// Export generates and outputs shell code for the current directory
func Export(params ExportParams) error {
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
	mergedConfig := loadAndMergeConfigs(chains.current, comps, log)

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

	// Get merged environment variables and aliases
	staticEnv, shellEnv := mergedConfig.GetEnvVars()
	aliases := mergedConfig.GetAliases()

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

// AllowParams contains parameters for the Allow command
type AllowParams struct {
	AuthPath    string
	PathToAllow string
	CachePath   string
	LogLevel    string
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

	fmt.Printf("Authorized: %s\n", params.PathToAllow)

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

	fmt.Printf("Revoked: %s\n", params.PathToRevoke)

	// Show cleanup tip if we're in the revoked directory
	if currentDir == params.PathToRevoke {
		fmt.Println("\n💡 Tip: Run 'cd ..' then 'cd -' to unload the Dirvana environment")
		fmt.Println("   Or run: 'eval \"$(dirvana export)\"' to reload the environment if you have parent configs")
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

// Init creates a sample .dirvana.yml config file in the current directory
func Init() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath := filepath.Join(currentDir, ".dirvana.yml")

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

	fmt.Printf("Created sample config: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit the config file to suit your needs")
	fmt.Println("  2. Run 'dirvana allow' to authorize this directory")
	fmt.Println("  3. Run 'dirvana setup' to install the shell hook")

	return nil
}

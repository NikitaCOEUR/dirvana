package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	dircontext "github.com/NikitaCOEUR/dirvana/internal/context"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
)

// ExportParams contains parameters for the Export command
type ExportParams struct {
	LogLevel  string
	PrevDir   string
	CachePath string
	AuthPath  string
}

// Export generates and outputs shell code for the current directory
func Export(params ExportParams) error {
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

	// Check if we need to cleanup previous context FIRST, before checking current dir config
	var cleanupCode string
	if dircontext.ShouldCleanup(params.PrevDir, currentDir) && params.PrevDir != "" {
		// Get previous context from cache to know what to cleanup
		if prevEntry, found := comps.cache.Get(params.PrevDir); found {
			cleanupCode = dircontext.GenerateCleanupCode(
				prevEntry.Aliases,
				prevEntry.Functions,
				prevEntry.EnvVars,
			)
			log.Debug().Str("prev_dir", params.PrevDir).Msg("Cleaning up previous context")
		}
	}

	// Find config files
	configFiles, err := config.FindConfigFiles(currentDir)
	if err != nil {
		// Don't fail the shell hook for this
		log.Debug().Err(err).Msg("Failed to find config files")
		// Still output cleanup code if needed
		fmt.Print(cleanupCode)
		return nil
	}

	if len(configFiles) == 0 {
		log.Debug().Msg("No config files found")
		// No config, but still output cleanup code if needed
		fmt.Print(cleanupCode)
		return nil
	}

	// Config file exists - check if directory is authorized
	allowed, err := comps.auth.IsAllowed(currentDir)
	if err != nil {
		return fmt.Errorf("failed to check authorization: %w", err)
	}

	if !allowed {
		// Config exists but directory not authorized - show clear warning
		log.Warn().
			Str("dir", currentDir).
			Msg("Dirvana config found but directory not authorized. Run: dirvana allow " + currentDir)
		fmt.Print("") // Output empty string so shell hook doesn't fail
		return nil
	}

	// Compute hash of the most specific config file
	mainConfig := configFiles[len(configFiles)-1]
	hash, err := comps.config.Hash(mainConfig)
	if err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}

	// Check cache
	if comps.cache.IsValid(currentDir, hash, version.Version) {
		entry, _ := comps.cache.Get(currentDir)
		log.Debug().Msg("Using cached shell code")
		fmt.Print(entry.ShellCode)
		return nil
	}

	// Load and merge configs
	merged, _, err := comps.config.LoadHierarchy(currentDir)
	if err != nil {
		return fmt.Errorf("failed to load config hierarchy: %w", err)
	}

	// Get static and shell-based env vars
	staticEnv, shellEnv := merged.GetEnvVars()

	// Get normalized aliases
	aliases := merged.GetAliases()

	// Track what we're defining for future cleanup (using helpers for efficiency)
	aliasKeys := keysFromAliasMap(aliases)
	functions := keysFromMap(merged.Functions)
	envVars := mergeTwoKeyLists(staticEnv, shellEnv)

	// Build command map for dirvana exec
	commandMap := buildCommandMap(aliases, merged.Functions)

	// Generate shell code
	shellCode := comps.shell.Generate(aliases, merged.Functions, staticEnv, shellEnv)

	// Prepend cleanup code if needed
	if cleanupCode != "" {
		shellCode = cleanupCode + "\n" + shellCode
	}

	// Update cache
	entry := &cache.Entry{
		Path:       currentDir,
		Hash:       hash,
		ShellCode:  shellCode,
		Timestamp:  time.Now(),
		Version:    version.Version,
		LocalOnly:  merged.LocalOnly,
		Aliases:    aliasKeys,
		Functions:  functions,
		EnvVars:    envVars,
		CommandMap: commandMap,
	}

	if err := comps.cache.Set(entry); err != nil {
		log.Warn().Err(err).Msg("Failed to update cache")
	}

	log.Debug().
		Str("hash", hash).
		Bool("local_only", merged.LocalOnly).
		Int("aliases", len(merged.Aliases)).
		Int("functions", len(merged.Functions)).
		Int("static_env", len(staticEnv)).
		Int("shell_env", len(shellEnv)).
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

# Shell aliases - Simple format (auto-detects completion)
aliases:
  ll: ls -lah
  gs: git status
  gd: git diff
  k: kubectl

  # Advanced format with completion control
  gp:
    command: git push
    completion: git  # Inherit git completion

  # Custom wrapper with inherited completion
  mykubectl:
    command: /usr/local/bin/kubectl-wrapper.sh
    completion: kubectl

  # Disable completion
  hello:
    command: echo "Hello World"
    completion: false

  # Custom completion (advanced)
  deploy:
    command: ./scripts/deploy.sh
    completion:
      bash: complete -W "dev staging prod" deploy
      zsh: compdef '_arguments "1: :(dev staging prod)"' deploy

# Shell functions
functions:
  mkcd: |
    mkdir -p "$1" && cd "$1"
  greet: |
    echo "Hello, $1!"

# Environment variables
env:
  PROJECT_NAME: myproject
  LOG_LEVEL: info

# Set to true to ignore parent configs
local_only: false
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

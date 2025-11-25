package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/errors"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
)

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
	log := logger.New(params.LogLevel, os.Stderr)

	authMgr, err := auth.New(params.AuthPath)
	if err != nil {
		return errors.NewAuthorizationError(params.PathToAllow, "failed to initialize auth", err)
	}

	// Check if already allowed - idempotent operation
	alreadyAllowed, err := authMgr.IsAllowed(params.PathToAllow)
	if err != nil {
		return errors.NewAuthorizationError(params.PathToAllow, "failed to check authorization", err)
	}
	if alreadyAllowed {
		log.Debug().Msg("already authorized: " + params.PathToAllow)
		return nil
	}

	if err := authMgr.Allow(params.PathToAllow); err != nil {
		return errors.NewAuthorizationError(params.PathToAllow, "failed to authorize", err)
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
			return errors.NewShellApprovalError(params.PathToAllow, "failed to auto-approve shell commands", err)
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
		return errors.NewAuthorizationError(params.PathToRevoke, "failed to initialize auth", err)
	}

	if err := authMgr.Revoke(params.PathToRevoke); err != nil {
		return errors.NewAuthorizationError(params.PathToRevoke, "failed to revoke", err)
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
		return errors.NewConfigurationError(path, "failed to load config", err)
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
		return errors.NewShellApprovalError(path, "failed to approve shell commands", err)
	}

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

// List displays all authorized directories
func List(authPath string) error {
	authMgr, err := auth.New(authPath)
	if err != nil {
		return errors.NewAuthorizationError("", "failed to initialize auth", err)
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

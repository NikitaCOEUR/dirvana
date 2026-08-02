package cli

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/config"
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
		return fmt.Errorf("failed to initialize auth: %w", err)
	}

	// Check if already allowed - idempotent operation
	alreadyAllowed, err := authMgr.IsAllowed(params.PathToAllow)
	if err != nil {
		return fmt.Errorf("failed to check authorization: %w", err)
	}
	if alreadyAllowed {
		log.Debug().Msg("already authorized: " + params.PathToAllow)
		return nil
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

	// Handle the env sh: commands consent in the same interaction instead
	// of surprising the user with a prompt in the middle of the next cd
	if err := handleShellApproval(params.PathToAllow, authMgr, params.AutoApproveShell, log); err != nil {
		return err
	}

	// If we're in the authorized directory, suggest loading the environment
	currentDir, err := os.Getwd()
	if err == nil && currentDir == params.PathToAllow {
		fmt.Println("\n💡 Tip: Run 'eval \"$(dirvana export)\"' to load the environment in your current shell")
		fmt.Println("\tOr run 'cd ..' then 'cd -' to reload the environment")
	}

	return nil
}

// handleShellApproval settles the consent for the env sh: commands that
// will run automatically on every cd. The commands are taken from the
// MERGED hierarchy (inherited ones included), because that is exactly what
// the export gate hashes and checks.
func handleShellApproval(path string, authMgr *auth.Auth, autoApprove bool, log *logger.Logger) error {
	shellEnv, err := mergedShellEnv(path, authMgr)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(shellEnv) == 0 {
		log.Debug().Msg("No shell commands found in config hierarchy")
		return nil
	}
	if !authMgr.RequiresShellApproval(path, shellEnv) {
		return nil
	}

	if autoApprove {
		if err := authMgr.ApproveShellCommands(path, shellEnv); err != nil {
			return fmt.Errorf("failed to auto-approve shell commands: %w", err)
		}
		fmt.Println("✓ Shell commands auto-approved")
		return nil
	}

	// Interactive consent, right now
	if err := displayShellCommandsForApproval(shellEnv); err != nil {
		return err
	}
	approved, err := promptShellApproval()
	if err != nil || !approved {
		// Not fatal: aliases and functions are usable, and the export
		// gate will ask again on the next cd
		fmt.Println("⚠️  Shell commands not approved - you will be asked again on the next cd")
		fmt.Println("   (or rerun with: dirvana allow --auto-approve-shell)")
		return nil
	}
	if err := authMgr.ApproveShellCommands(path, shellEnv); err != nil {
		return fmt.Errorf("failed to approve shell commands: %w", err)
	}
	fmt.Println("✓ Shell commands approved")
	return nil
}

// mergedShellEnv returns the env sh: commands effective in path, i.e. the
// merged hierarchy as the export gate sees it
func mergedShellEnv(path string, authMgr *auth.Auth) (map[string]string, error) {
	merged, _, err := config.New().LoadHierarchyWithAuth(path, authMgr)
	if err != nil {
		return nil, err
	}
	if merged == nil {
		return nil, nil
	}
	_, shellEnv := merged.GetEnvVars()
	return shellEnv, nil
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

// Display dynamic shell commands for approval. The message goes through
// notifyUser so it reaches the terminal even inside $(dirvana export).
func displayShellCommandsForApproval(shellEnv map[string]string) error {
	if len(shellEnv) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("\n⚠️  This configuration contains dynamic shell commands:\n\n")
	keys := make([]string, 0, len(shellEnv))
	for k := range shellEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "   • %s: %s\n", key, shellEnv[key])
	}
	b.WriteString("\nThese commands will execute to set environment variables.\n")
	notifyUser(b.String())
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

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
)

// ExecParams contains parameters for the Exec command
type ExecParams struct {
	CachePath string
	AuthPath  string
	LogLevel  string
	Alias     string
	Args      []string
}

// Exec resolves and executes an alias or function defined by Dirvana
func Exec(params ExecParams) error {
	log := logger.New(params.LogLevel, os.Stderr)

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get merged command maps from the full hierarchy
	// This respects global config, ignore_global, local_only, and authorization
	commandMap, _, err := getMergedCommandMaps(currentDir, params.CachePath, params.AuthPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(commandMap) == 0 {
		return fmt.Errorf("no dirvana context found for alias '%s'", params.Alias)
	}

	// Check if alias exists in this context
	command, found := commandMap[params.Alias]
	if !found {
		return fmt.Errorf("alias '%s' not found in dirvana context", params.Alias)
	}

	log.Debug().Str("alias", params.Alias).Str("command", command).Msg("Resolving alias")

	// Execute via shell to allow variable expansion, pipes, redirections, etc.
	// Detect which shell to use
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash" // Fallback to bash (will be found via PATH)
	}

	// Find shell executable path
	execPath, err := exec.LookPath(shell)
	if err != nil {
		return fmt.Errorf("shell not found: %s", shell)
	}

	// Build argv for shell execution
	// Use: shell -c 'command "$@"' shell args...
	// The first arg after the command becomes $0 (we use shell name)
	// The remaining args become $1, $2, $3, etc. which are captured by "$@"
	var argv []string
	if len(params.Args) > 0 {
		// Append "$@" to command to receive user arguments
		argv = []string{shell, "-c", command + ` "$@"`, shell}
		argv = append(argv, params.Args...)
	} else {
		// No user arguments, just execute the command
		argv = []string{shell, "-c", command}
	}

	log.Debug().
		Str("shell", shell).
		Str("argv", fmt.Sprintf("%q", argv)).
		Msg("Executing command via shell")

	// Execute the command via shell (replace current process)
	// This allows shell variable expansion, pipes, redirections, etc.
	err = syscall.Exec(execPath, argv, os.Environ())

	// If we reach here, syscall.Exec failed (extremely rare)
	return fmt.Errorf("failed to execute command: %w", err)
}

// findCacheEntry searches for a cache entry in the current directory or parent directories
func findCacheEntry(c *cache.Cache, dir string) (*cache.Entry, bool) {
	dir = filepath.Clean(dir)

	// Try current directory first
	if entry, found := c.Get(dir); found {
		return entry, true
	}

	// Walk up the directory tree
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent

		if entry, found := c.Get(dir); found {
			return entry, true
		}
	}

	return nil, false
}

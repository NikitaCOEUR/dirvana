package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
)

// ExecParams contains parameters for the Exec command
type ExecParams struct {
	CachePath string
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

	// Load cache
	c, err := cache.New(params.CachePath)
	if err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	// Find the cache entry for current directory or parent directories
	entry, found := findCacheEntry(c, currentDir)
	if !found {
		return fmt.Errorf("no dirvana context found for alias '%s'", params.Alias)
	}

	// Check if alias exists in this context
	command, found := entry.CommandMap[params.Alias]
	if !found {
		return fmt.Errorf("alias '%s' not found in dirvana context", params.Alias)
	}

	log.Debug().Str("alias", params.Alias).Str("command", command).Msg("Resolving alias")

	// Parse command (handle multi-word commands)
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return fmt.Errorf("empty command for alias '%s'", params.Alias)
	}

	// Combine command parts with user args
	allArgs := append(cmdParts[1:], params.Args...)

	// Find executable path
	execPath, err := exec.LookPath(cmdParts[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", cmdParts[0])
	}

	// Prepare arguments for exec (argv[0] should be the program name)
	argv := append([]string{cmdParts[0]}, allArgs...)

	// Execute the command directly (replace current process)
	// This is the most efficient way - no fork overhead
	// NOTE: If this returns, it means exec failed (should never happen after LookPath succeeds)
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

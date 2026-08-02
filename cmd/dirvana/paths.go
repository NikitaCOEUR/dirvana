package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// appPaths holds the XDG-resolved state file locations
type appPaths struct {
	cache string
	auth  string
}

// resolvePaths computes the cache and auth file paths from XDG directories
func resolvePaths() appPaths {
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, _ := os.UserHomeDir()
		cacheHome = filepath.Join(home, ".cache")
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}

	return appPaths{
		cache: filepath.Join(cacheHome, "dirvana", "cache.json"),
		auth:  filepath.Join(dataHome, "dirvana", "authorized_v2.json"),
	}
}

// targetPathOrCwd returns the first positional argument resolved to an
// absolute path, or the current directory when no argument is given
func targetPathOrCwd(arg string) (string, error) {
	if arg == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		return currentDir, nil
	}

	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	return abs, nil
}

package cli

import (
	"fmt"
	"os"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
)

// CleanParams holds parameters for the Clean function
type CleanParams struct {
	CachePath string
	LogLevel  string
	All       bool
}

// Clean removes cache entries
func Clean(params CleanParams) error {
	log := logger.New(params.LogLevel, nil)

	// Initialize cache
	c, err := cache.New(params.CachePath)
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	if params.All {
		// Clear entire cache
		if err := c.Clear(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		log.Info().Msg("All cache entries cleared")
		fmt.Println("✓ All cache entries cleared")
	} else {
		// Clear hierarchy for current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		if err := c.ClearHierarchy(currentDir); err != nil {
			return fmt.Errorf("failed to clear cache hierarchy: %w", err)
		}
		log.Info().Str("dir", currentDir).Msg("Cache cleared for directory hierarchy")
		fmt.Printf("✓ Cache cleared for %s and its hierarchy\n", currentDir)
	}

	return nil
}

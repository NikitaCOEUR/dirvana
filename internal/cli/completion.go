package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
)

// CompletionParams contains parameters for the Completion command
type CompletionParams struct {
	CachePath string
	LogLevel  string
	Line      string   // Full command line being completed
	Point     int      // Cursor position in the line
	Words     []string // Words in the command line (COMP_WORDS)
	CWord     int      // Index of word being completed (COMP_CWORD)
}

// Completion generates shell completions for dirvana-managed aliases
func Completion(params CompletionParams) error {
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

	// Find the cache entry
	entry, found := findCacheEntry(c, currentDir)
	if !found {
		// No context, no completions
		return nil
	}

	// First word is the alias being executed
	if len(params.Words) == 0 {
		return nil
	}

	aliasName := params.Words[0]

	// Look up the actual command
	command, found := entry.CommandMap[aliasName]
	if !found {
		return nil
	}

	log.Debug().
		Str("alias", aliasName).
		Str("command", command).
		Int("cword", params.CWord).
		Msg("Generating completion")

	// For functions, we don't have completions yet
	if strings.HasPrefix(command, "__dirvana_function__") {
		return nil
	}

	// Parse the base command
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return nil
	}

	baseCmd := cmdParts[0]

	// Try to delegate to the base command's completion
	// We'll use bash's complete -p to check if completion exists
	suggestions, err := getCompletions(baseCmd, params.Words[1:], params.CWord-1)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get completions")
		return nil
	}

	// Print suggestions (one per line)
	for _, suggestion := range suggestions {
		fmt.Println(suggestion)
	}

	return nil
}

// getCompletions tries to get completions for a command
// This is a simplified version - a full implementation would need to:
// 1. Check if the command has bash-completion or zsh-completion
// 2. Source the completion script
// 3. Invoke the completion function
// For now, we just return empty (basic file/directory completion will be used)
func getCompletions(baseCmd string, args []string, cword int) ([]string, error) {
	// TODO: Implement proper completion delegation
	// For now, we'll just check if we can execute the command with --help or similar
	// This is a placeholder that returns no suggestions
	// The shell will fall back to default completion (files/directories)

	// Check if command exists
	_, err := exec.LookPath(baseCmd)
	if err != nil {
		return nil, err
	}

	// Return empty for now - shell will use default completion
	return []string{}, nil
}

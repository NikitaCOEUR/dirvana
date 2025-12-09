package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/completion"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
	"github.com/NikitaCOEUR/dirvana/internal/trace"
)

// CompletionParams contains parameters for the Completion command
type CompletionParams struct {
	CachePath string
	AuthPath  string
	LogLevel  string
	Words     []string // Words in the command line (COMP_WORDS)
	CWord     int      // Index of word being completed (COMP_CWORD)
}

// Completion generates shell completions for dirvana-managed aliases
// This is called by the shell completion function with the current command line state
func Completion(params CompletionParams) error {
	ctx := context.Background()
	defer trace.Region(ctx, "cli.Completion")()

	log := logger.New(params.LogLevel, os.Stderr)

	// Validate input
	if len(params.Words) == 0 {
		return nil
	}

	// First word is the alias being executed
	aliasName := params.Words[0]

	log.Debug().
		Str("alias", aliasName).
		Int("words_count", len(params.Words)).
		Int("cword", params.CWord).
		Str("words", fmt.Sprintf("%q", params.Words)).
		Msg("Received completion request")

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get merged command maps from the full hierarchy
	// This respects global config, ignore_global, local_only, and authorization
	var commandMap, completionMap map[string]string
	trace.WithRegion(ctx, "getMergedCommandMaps", func() {
		commandMap, completionMap, err = getMergedCommandMaps(currentDir, params.CachePath, params.AuthPath)
	})
	if err != nil {
		// Failed to load config, no completions
		return nil
	}

	if len(commandMap) == 0 {
		// No dirvana context, no completions
		return nil
	}

	// Look up the actual command for this alias
	command, found := commandMap[aliasName]
	if !found {
		// Not a dirvana-managed alias
		return nil
	}

	// Check if there's a completion override
	// (e.g., k -> kubecolor for exec, but kubectl for completion)
	completionCmd := command
	if completionMap != nil {
		if override, ok := completionMap[aliasName]; ok {
			completionCmd = override
		}
	}

	log.Debug().
		Str("alias", aliasName).
		Str("command", command).
		Str("completion_cmd", completionCmd).
		Msg("Resolving completion command")

	// For functions, we don't have smart completions
	if strings.HasPrefix(command, "__dirvana_function__") {
		return nil
	}

	// Parse the base command
	cmdParts := strings.Fields(completionCmd)
	if len(cmdParts) == 0 {
		return nil
	}

	baseCmd := cmdParts[0]

	// Create completion engine early to check detection cache
	// If we have a cache hit, we know the command exists (was working < 24h ago)
	cacheDir := filepath.Dir(params.CachePath)
	engine := completion.NewEngine(cacheDir)

	// Check if command exists - skip LookPath if we have a detection cache hit
	// (if tool is in detection cache, it was working recently)
	if !engine.HasCachedDetection(baseCmd) {
		var lookPathErr error
		trace.WithRegion(ctx, "exec.LookPath", func() {
			_, lookPathErr = exec.LookPath(baseCmd)
		})
		if lookPathErr != nil {
			log.Debug().Str("cmd", baseCmd).Msg("Command not found, no completion")
			return nil
		}
	} else {
		log.Debug().Str("cmd", baseCmd).Msg("Skipping LookPath (detection cache hit)")
	}

	// Prepare arguments for completion
	args := params.Words[1:] // Remove the alias name (first word)

	// Cobra/kubectl and other tools expect at least one argument for completion
	// If args is empty (only typed the command), add an empty string to get subcommand completions
	if len(args) == 0 {
		args = append(args, "")
		log.Debug().Msg("Added empty arg for initial completion")
	}

	// If COMP_CWORD points beyond existing words, we're completing a new empty word
	if params.CWord >= len(params.Words) {
		// Only add if we haven't already added one above
		if len(args) > 0 && args[len(args)-1] != "" {
			args = append(args, "")
			log.Debug().Int("cword", params.CWord).Int("words_len", len(params.Words)).Msg("Added empty word for completion beyond last word")
		}
	}

	// Get current word being completed
	var currentWord string
	if params.CWord > 0 && params.CWord < len(params.Words) {
		currentWord = params.Words[params.CWord]
	}

	log.Debug().
		Str("base_cmd", baseCmd).
		Str("current_word", currentWord).
		Int("args_count", len(args)).
		Msg("Starting completion")

	// Get suggestions from the completion engine (already created above)
	var result *completion.Result
	trace.WithRegion(ctx, "engine.Complete", func() {
		result, err = engine.Complete(baseCmd, args)
	})
	if err != nil {
		log.Debug().Err(err).Msg("Completion failed")
		return nil
	}

	log.Debug().
		Int("suggestions_count", len(result.Suggestions)).
		Str("source", result.Source).
		Msg("Got completions")

	// Filter suggestions by current word prefix
	filtered := engine.Filter(result.Suggestions, currentWord)

	log.Debug().
		Int("filtered_count", len(filtered)).
		Str("prefix", currentWord).
		Msg("Filtered completions")

	// Sort suggestions alphabetically for stable ordering
	// Some tools (like packer) return suggestions in random order
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Value < filtered[j].Value
	})

	// Output suggestions in the format: value\tdescription
	// Shell will parse this and show both value and description
	for _, suggestion := range filtered {
		if suggestion.Description != "" {
			fmt.Printf("%s\t%s\n", suggestion.Value, suggestion.Description)
		} else {
			fmt.Println(suggestion.Value)
		}
	}

	return nil
}

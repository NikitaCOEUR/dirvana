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

// resolveCompletionCommand looks up the actual command for an alias and its completion override
func resolveCompletionCommand(aliasName, currentDir, cachePath, authPath string, log *logger.Logger) (command, completionCmd string, err error) {
	ctx := context.Background()

	// Get merged command maps from the full hierarchy
	var commandMap, completionMap map[string]string
	trace.WithRegion(ctx, "getMergedCommandMaps", func() {
		commandMap, completionMap, err = getMergedCommandMaps(currentDir, cachePath, authPath)
	})
	if err != nil {
		return "", "", err
	}

	if len(commandMap) == 0 {
		return "", "", fmt.Errorf("no dirvana context")
	}

	// Look up the actual command for this alias
	var found bool
	command, found = commandMap[aliasName]
	if !found {
		return "", "", fmt.Errorf("not a dirvana-managed alias")
	}

	// Check if there's a completion override
	completionCmd = command
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

	return command, completionCmd, nil
}

// prepareCompletionArgs prepares the arguments for completion based on shell state
func prepareCompletionArgs(params CompletionParams, log *logger.Logger) []string {
	args := params.Words[1:] // Remove the alias name (first word)

	// Cobra/kubectl and other tools expect at least one argument for completion
	if len(args) == 0 {
		args = append(args, "")
		log.Debug().Msg("Added empty arg for initial completion")
		return args
	}

	// If COMP_CWORD points beyond existing words, we're completing a new empty word
	if params.CWord >= len(params.Words) && args[len(args)-1] != "" {
		args = append(args, "")
		log.Debug().Int("cword", params.CWord).Int("words_len", len(params.Words)).Msg("Added empty word for completion beyond last word")
	}

	return args
}

// getCurrentWord returns the word currently being completed
func getCurrentWord(params CompletionParams) string {
	if params.CWord > 0 && params.CWord < len(params.Words) {
		return params.Words[params.CWord]
	}
	return ""
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

	// Resolve the command and completion command for this alias
	command, completionCmd, err := resolveCompletionCommand(aliasName, currentDir, params.CachePath, params.AuthPath, log)
	if err != nil {
		// Failed to resolve or not a dirvana alias
		return nil
	}

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

	// Create completion engine and verify command exists
	cacheDir := filepath.Dir(params.CachePath)
	engine := completion.NewEngine(cacheDir)

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

	// Prepare completion arguments
	args := prepareCompletionArgs(params, log)
	currentWord := getCurrentWord(params)

	log.Debug().
		Str("base_cmd", baseCmd).
		Str("current_word", currentWord).
		Int("args_count", len(args)).
		Msg("Starting completion")

	// Get suggestions from the completion engine
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

	// Filter and sort suggestions
	filtered := engine.Filter(result.Suggestions, currentWord)

	log.Debug().
		Int("filtered_count", len(filtered)).
		Str("prefix", currentWord).
		Msg("Filtered completions")

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Value < filtered[j].Value
	})

	// Output suggestions
	for _, suggestion := range filtered {
		if suggestion.Description != "" {
			fmt.Printf("%s\t%s\n", suggestion.Value, suggestion.Description)
		} else {
			fmt.Println(suggestion.Value)
		}
	}

	return nil
}

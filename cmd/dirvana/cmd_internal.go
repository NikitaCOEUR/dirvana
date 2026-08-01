package main

import (
	"context"
	"fmt"
	"os"

	dircli "github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/urfave/cli/v3"
)

// newExecCmd builds the hidden command wrapped by every generated alias
func newExecCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:            "exec",
		Usage:           "Execute a dirvana-managed alias or function",
		ArgsUsage:       "<alias> [args...]",
		Hidden:          true, // Hidden from help - used internally by shell aliases
		SkipFlagParsing: true, // Don't parse flags - pass them directly to the wrapped command
		HideHelp:        true, // Don't show help for this internal command
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() == 0 {
				return fmt.Errorf("alias name required")
			}

			alias := cmd.Args().Get(0)
			args := cmd.Args().Slice()[1:]

			return dircli.Exec(dircli.ExecParams{
				CachePath: paths.cache,
				AuthPath:  paths.auth,
				LogLevel:  cmd.String("log-level"),
				Alias:     alias,
				Args:      args,
			})
		},
	}
}

// newCompletionCmd builds the hidden command called by the generated shell
// completion functions
func newCompletionCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:            "completion",
		Usage:           "Generate shell completions for dirvana-managed aliases",
		ArgsUsage:       "[completion-args...]",
		Hidden:          true, // Hidden from help - used internally by completion functions
		SkipFlagParsing: true, // Don't parse flags - pass them directly to the wrapped command
		HideHelp:        true, // Don't show help for this internal command
		Action: func(_ context.Context, cmd *cli.Command) error {
			// Bash completion provides COMP_WORDS via args
			// and COMP_CWORD via DIRVANA_COMP_CWORD env var
			words := completionWords(os.Args)

			// Get COMP_CWORD from environment
			cword := len(words) - 1 // default to last word
			if cwordStr := os.Getenv("DIRVANA_COMP_CWORD"); cwordStr != "" {
				_, _ = fmt.Sscanf(cwordStr, "%d", &cword) // Ignore errors, keep default
			}

			return dircli.Completion(dircli.CompletionParams{
				CachePath: paths.cache,
				AuthPath:  paths.auth,
				LogLevel:  cmd.String("log-level"),
				Words:     words,
				CWord:     cword,
			})
		},
	}
}

// completionWords extracts the completion words from the raw process args.
// IMPORTANT: os.Args is used instead of cmd.Args() because urfave/cli
// treats "--" as a special separator and filters it out, but we need it
// for kubectl-style completion.
func completionWords(rawArgs []string) []string {
	var words []string
	foundCompletion := false
	skipFirstDoubleDash := true
	for _, arg := range rawArgs {
		if arg == "completion" {
			foundCompletion = true
			continue
		}
		if foundCompletion {
			// Skip the first "--" which is just bash's separator
			// but keep subsequent "--" as they might be meaningful (e.g., kubectl -- ...)
			if arg == "--" && skipFirstDoubleDash {
				skipFirstDoubleDash = false
				continue
			}
			words = append(words, arg)
		}
	}
	return words
}

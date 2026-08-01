package completion

import (
	"context"
	"slices"
)

// FlagCompleter handles tools that use --generate-shell-completion flag
// This is used by tools built with github.com/urfave/cli and similar frameworks
type FlagCompleter struct{}

// NewFlagCompleter creates a new flag-based completer
func NewFlagCompleter() *FlagCompleter {
	return &FlagCompleter{}
}

// Supports checks if the tool supports --generate-shell-completion
// We verify by checking that it returns a simple list of words
func (f *FlagCompleter) Supports(tool string, _ []string) bool {
	// Test if tool accepts --generate-shell-completion (with timeout)
	ctx := context.Background()
	output, err := execWithTimeout(ctx, tool, "--generate-shell-completion")

	// If command failed or returned nothing, doesn't support
	if err != nil || len(output) == 0 {
		return false
	}

	// Use common validator to check if output looks like suggestions
	return validateSimpleOutput(output, 1)
}

// Complete uses --generate-shell-completion to get suggestions
func (f *FlagCompleter) Complete(tool string, args []string) ([]Suggestion, error) {
	// Build command: tool [args...] --generate-shell-completion
	// Note: we pass all args INCLUDING the current word being completed.
	// The caller-owned slice is shared with other completers running
	// concurrently in the engine, so never append to it in place.
	cmdArgs := slices.Concat(args, []string{"--generate-shell-completion"})

	ctx := context.Background()
	output, err := execWithTimeout(ctx, tool, cmdArgs...)
	if err != nil {
		return nil, err
	}

	// One suggestion per line, no descriptions
	return parseCompletionOutput(output, false), nil
}

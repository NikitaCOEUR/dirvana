package completion

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
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
	// Test if tool accepts --generate-shell-completion
	cmd := exec.Command(tool, "--generate-shell-completion")
	output, err := cmd.Output()

	// If command failed or returned nothing, doesn't support
	if err != nil || len(output) == 0 {
		return false
	}

	// Check if output looks like a simple list of commands (not help text)
	// urfave/cli returns simple words, one per line
	// Help text usually has multiple words per line or special characters
	lines := strings.Split(string(output), "\n")
	validLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Valid suggestion: single word without spaces or special chars
		// (allow hyphens and underscores for command names)
		words := strings.Fields(line)
		if len(words) == 1 {
			validLines++
		}
	}

	// Must have at least one valid suggestion
	return validLines > 0
}

// Complete uses --generate-shell-completion to get suggestions
func (f *FlagCompleter) Complete(tool string, args []string) ([]Suggestion, error) {
	// Build command: tool [args...] --generate-shell-completion
	// Note: we pass all args INCLUDING the current word being completed
	cmdArgs := append(args, "--generate-shell-completion")

	cmd := exec.Command(tool, cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseFlagOutput(output), nil
}

// parseFlagOutput parses flag-based completion output
// Format: one suggestion per line, no descriptions (simple list)
func parseFlagOutput(output []byte) []Suggestion {
	var suggestions []Suggestion

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		suggestions = append(suggestions, Suggestion{
			Value:       line,
			Description: "",
		})
	}

	return suggestions
}

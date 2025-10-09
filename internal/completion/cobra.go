package completion

import (
	"context"
	"fmt"
	"strings"
)

// CobraCompleter handles completion for Cobra-based CLIs (kubectl, helm, etc.)
type CobraCompleter struct{}

// NewCobraCompleter creates a new Cobra completer
func NewCobraCompleter() *CobraCompleter {
	return &CobraCompleter{}
}

// Supports checks if the tool supports Cobra's __complete API
// We verify by checking for Cobra's directive format in the output
func (c *CobraCompleter) Supports(tool string, _ []string) bool {
	// Try calling tool __complete with empty arg (with timeout)
	ctx := context.Background()
	output, err := execWithTimeout(ctx, tool, "__complete", "")

	// If command failed or returned nothing, doesn't support
	if err != nil || len(output) == 0 {
		return false
	}

	// Check if output contains a Cobra directive (line starting with ":")
	// Cobra always outputs a directive like ":4" or ":0" at the end
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 1 && line[0] == ':' {
			// Verify it looks like a number after the colon
			if _, err := fmt.Sscanf(line, ":%d", new(int)); err == nil {
				return true
			}
		}
	}

	return false
}

// Complete executes the tool's __complete command and parses the output
func (c *CobraCompleter) Complete(tool string, args []string) ([]Suggestion, error) {
	// Build the __complete command
	// tool __complete <args...>
	completeArgs := append([]string{"__complete"}, args...)

	ctx := context.Background()
	output, err := execWithTimeout(ctx, tool, completeArgs...)
	if err != nil {
		return nil, err
	}

	return parseCobraOutput(output), nil
}

// parseCobraOutput parses Cobra completion output format:
// - Lines with format: "value\tdescription"
// - Lines starting with ":" are directives (ignore them)
// - Empty lines are ignored
func parseCobraOutput(output []byte) []Suggestion {
	// First, filter out Cobra directives and help messages
	var filteredLines []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip Cobra directives (lines starting with ":") and help messages
		if strings.HasPrefix(line, ":") || strings.Contains(line, "Completion ended") || strings.Contains(line, "ShellCompDirective") {
			continue
		}

		filteredLines = append(filteredLines, line)
	}

	// Use common parser with description support
	filteredOutput := []byte(strings.Join(filteredLines, "\n"))
	return parseCompletionOutput(filteredOutput, true)
}

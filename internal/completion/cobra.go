package completion

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
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
	// Try calling tool __complete with empty arg
	cmd := exec.Command(tool, "__complete", "")
	output, err := cmd.Output()

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
	cmd := exec.Command(tool, completeArgs...)

	output, err := cmd.Output()
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
	var suggestions []Suggestion
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip Cobra directives (lines starting with ":") and help messages
		if strings.HasPrefix(line, ":") || strings.Contains(line, "Completion ended") || strings.Contains(line, "ShellCompDirective") {
			continue
		}

		// Split by tab to separate value from description
		parts := strings.SplitN(line, "\t", 2)
		suggestion := Suggestion{
			Value: parts[0],
		}

		// Add description if present
		if len(parts) > 1 {
			suggestion.Description = parts[1]
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

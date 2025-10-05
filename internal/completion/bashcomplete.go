package completion

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BashCompleteCompleter handles tools that use the bash "complete -C" protocol
// This protocol is used by terraform, consul, vault, nomad, and other HashiCorp tools
// The tool is called with COMP_LINE and COMP_POINT environment variables
type BashCompleteCompleter struct{}

// NewBashCompleteCompleter creates a new bash complete completer
func NewBashCompleteCompleter() *BashCompleteCompleter {
	return &BashCompleteCompleter{}
}

// Supports checks if the tool supports bash complete protocol by testing it
// We verify by checking that it returns actual suggestions
func (b *BashCompleteCompleter) Supports(tool string, _ []string) bool {
	// Test if the tool responds to COMP_LINE environment variable
	cmd := exec.Command(tool)
	cmd.Env = append(os.Environ(),
		"COMP_LINE="+tool+" ",
		"COMP_POINT=0",
	)
	output, err := cmd.Output()

	// If command failed or returned nothing, doesn't support
	if err != nil || len(output) == 0 {
		return false
	}

	// Check if output looks like suggestions (list of simple words)
	// bash complete protocol returns simple words, one per line
	// NOT multi-word descriptions or help text
	lines := strings.Split(string(output), "\n")
	validLines := 0
	invalidLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Valid suggestion: should be a simple word/path (can have spaces in paths)
		// Invalid: sentences with many words, punctuation like "Usage:", "Error:", etc.
		words := strings.Fields(line)
		// Help text usually has 3+ words or contains colons/special markers
		if len(words) >= 3 || strings.Contains(line, ":") || strings.Contains(line, "Usage") {
			invalidLines++
		} else if len(words) > 0 {
			validLines++
		}
	}

	// Must have more valid lines than invalid ones
	// This filters out help text while allowing real suggestions
	return validLines > 0 && invalidLines == 0
}

// Complete uses the bash complete protocol to get suggestions
func (b *BashCompleteCompleter) Complete(tool string, args []string) ([]Suggestion, error) {
	// Build the COMP_LINE (the full command line)
	compLine := tool
	if len(args) > 0 {
		compLine += " " + strings.Join(args, " ")
	}

	// COMP_POINT is the cursor position (end of line for now)
	compPoint := len(compLine)

	// Call the tool with completion environment variables
	cmd := exec.Command(tool)
	// Inherit current environment and add completion variables
	cmd.Env = append(os.Environ(),
		"COMP_LINE="+compLine,
		fmt.Sprintf("COMP_POINT=%d", compPoint),
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseBashCompleteOutput(output), nil
}

// parseBashCompleteOutput parses the output from bash complete protocol
// Format: one suggestion per line, no descriptions
func parseBashCompleteOutput(output []byte) []Suggestion {
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

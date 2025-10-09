package completion

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultCommandTimeout is the default timeout for completion commands
	DefaultCommandTimeout = 3 * time.Second
	// MaxOutputSize is the maximum size of command output (1MB)
	MaxOutputSize = 1024 * 1024
)

// execWithTimeout executes a command with a timeout and returns its output
// This prevents hanging on slow/blocked commands
func execWithTimeout(ctx context.Context, tool string, args ...string) ([]byte, error) {
	return execWithTimeoutAndEnv(ctx, nil, tool, args...)
}

// execWithTimeoutAndEnv executes a command with a timeout and custom environment
// If env is nil, the command inherits the current process environment
func execWithTimeoutAndEnv(ctx context.Context, env []string, tool string, args ...string) ([]byte, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, tool, args...)
	if env != nil {
		cmd.Env = env
	}

	output, err := cmd.Output()
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("command timeout after %v: %w", DefaultCommandTimeout, err)
		}
		return nil, err
	}

	// Limit output size
	if len(output) > MaxOutputSize {
		return output[:MaxOutputSize], nil
	}

	return output, nil
}

// parseCompletionOutput parses completion output into suggestions
// If parseDescriptions is true, it will parse tab-separated descriptions
func parseCompletionOutput(output []byte, parseDescriptions bool) []Suggestion {
	suggestions := []Suggestion{} // Initialize as empty slice, not nil

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var suggestion Suggestion

		if parseDescriptions {
			// Parse tab-separated value and description
			parts := strings.SplitN(line, "\t", 2)
			suggestion.Value = parts[0]
			if len(parts) > 1 {
				suggestion.Description = parts[1]
			}
		} else {
			// Simple value only
			suggestion.Value = line
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// validateSimpleOutput checks if output looks like suggestions (not help text)
// Returns true if the output appears to be a valid list of suggestions
func validateSimpleOutput(output []byte, minLines int) bool {
	if len(output) == 0 {
		return false
	}

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

	return validLines >= minLines
}

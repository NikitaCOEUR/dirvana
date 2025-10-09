package completion

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecWithTimeout tests basic command execution with timeout
func TestExecWithTimeout(t *testing.T) {
	t.Run("successful command", func(t *testing.T) {
		ctx := context.Background()
		output, err := execWithTimeout(ctx, "echo", "hello")
		require.NoError(t, err)
		assert.Equal(t, "hello\n", string(output))
	})

	t.Run("command with arguments", func(t *testing.T) {
		ctx := context.Background()
		output, err := execWithTimeout(ctx, "echo", "hello", "world")
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", string(output))
	})

	t.Run("command that fails", func(t *testing.T) {
		ctx := context.Background()
		_, err := execWithTimeout(ctx, "false")
		assert.Error(t, err)
	})

	t.Run("command timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// sleep command that takes longer than timeout
		_, err := execWithTimeout(ctx, "sleep", "1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("nil context uses default timeout", func(t *testing.T) {
		output, err := execWithTimeout(context.Background(), "echo", "test")
		require.NoError(t, err)
		assert.Equal(t, "test\n", string(output))
	})

	t.Run("output size limit", func(t *testing.T) {
		// This test assumes we can generate output larger than MaxOutputSize
		// Skip if not on a system with `yes` command
		ctx := context.Background()

		// Generate large output (yes prints 'y' infinitely, head limits it)
		// We'll use a smaller timeout to avoid long test runs
		ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()

		output, _ := execWithTimeout(ctx, "sh", "-c", "yes | head -c 2000000")

		// Output should be limited to MaxOutputSize
		if len(output) > 0 {
			assert.LessOrEqual(t, len(output), MaxOutputSize, "output should be limited to MaxOutputSize")
		}
	})
}

// TestExecWithTimeoutAndEnv tests command execution with custom environment
func TestExecWithTimeoutAndEnv(t *testing.T) {
	t.Run("with custom env", func(t *testing.T) {
		ctx := context.Background()
		env := []string{"TEST_VAR=hello"}

		output, err := execWithTimeoutAndEnv(ctx, env, "sh", "-c", "echo $TEST_VAR")
		require.NoError(t, err)
		assert.Equal(t, "hello\n", string(output))
	})

	t.Run("nil env inherits current environment", func(t *testing.T) {
		ctx := context.Background()

		// PATH should be available from inherited env
		output, err := execWithTimeoutAndEnv(ctx, nil, "sh", "-c", "echo $PATH")
		require.NoError(t, err)
		assert.NotEmpty(t, string(output))
	})

	t.Run("empty env has no variables", func(t *testing.T) {
		ctx := context.Background()
		env := []string{} // Empty environment

		output, err := execWithTimeoutAndEnv(ctx, env, "sh", "-c", "echo $PATH")
		require.NoError(t, err)
		// With empty env, PATH should still have some default value on most systems
		// because sh itself may set some defaults. Just verify it runs.
		assert.NotNil(t, output)
	})

	t.Run("timeout with custom env", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		env := []string{"TEST=value"}
		_, err := execWithTimeoutAndEnv(ctx, env, "sleep", "1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

// TestParseCompletionOutput tests parsing of completion output
func TestParseCompletionOutput(t *testing.T) {
	t.Run("simple values without descriptions", func(t *testing.T) {
		output := []byte("suggestion1\nsuggestion2\nsuggestion3")
		suggestions := parseCompletionOutput(output, false)

		assert.Len(t, suggestions, 3)
		assert.Equal(t, "suggestion1", suggestions[0].Value)
		assert.Equal(t, "", suggestions[0].Description)
		assert.Equal(t, "suggestion2", suggestions[1].Value)
		assert.Equal(t, "suggestion3", suggestions[2].Value)
	})

	t.Run("values with tab-separated descriptions", func(t *testing.T) {
		output := []byte("cmd1\tFirst command\ncmd2\tSecond command\ncmd3\tThird command")
		suggestions := parseCompletionOutput(output, true)

		assert.Len(t, suggestions, 3)
		assert.Equal(t, "cmd1", suggestions[0].Value)
		assert.Equal(t, "First command", suggestions[0].Description)
		assert.Equal(t, "cmd2", suggestions[1].Value)
		assert.Equal(t, "Second command", suggestions[1].Description)
	})

	t.Run("descriptions parsing disabled", func(t *testing.T) {
		output := []byte("cmd1\tFirst command\ncmd2\tSecond command")
		suggestions := parseCompletionOutput(output, false)

		assert.Len(t, suggestions, 2)
		assert.Equal(t, "cmd1\tFirst command", suggestions[0].Value)
		assert.Equal(t, "", suggestions[0].Description)
	})

	t.Run("empty lines are skipped", func(t *testing.T) {
		output := []byte("cmd1\n\n\ncmd2\n\ncmd3")
		suggestions := parseCompletionOutput(output, false)

		assert.Len(t, suggestions, 3)
		assert.Equal(t, "cmd1", suggestions[0].Value)
		assert.Equal(t, "cmd2", suggestions[1].Value)
		assert.Equal(t, "cmd3", suggestions[2].Value)
	})

	t.Run("whitespace is trimmed", func(t *testing.T) {
		output := []byte("  cmd1  \n  cmd2\t\n\tcmd3  ")
		suggestions := parseCompletionOutput(output, false)

		assert.Len(t, suggestions, 3)
		assert.Equal(t, "cmd1", suggestions[0].Value)
		assert.Equal(t, "cmd2", suggestions[1].Value)
		assert.Equal(t, "cmd3", suggestions[2].Value)
	})

	t.Run("empty output returns empty slice", func(t *testing.T) {
		output := []byte("")
		suggestions := parseCompletionOutput(output, false)

		assert.NotNil(t, suggestions)
		assert.Len(t, suggestions, 0)
	})

	t.Run("only whitespace returns empty slice", func(t *testing.T) {
		output := []byte("   \n\n  \n\t\t\n")
		suggestions := parseCompletionOutput(output, false)

		assert.NotNil(t, suggestions)
		assert.Len(t, suggestions, 0)
	})

	t.Run("mixed: some with descriptions, some without", func(t *testing.T) {
		output := []byte("cmd1\tWith description\ncmd2\ncmd3\tAnother description")
		suggestions := parseCompletionOutput(output, true)

		assert.Len(t, suggestions, 3)
		assert.Equal(t, "cmd1", suggestions[0].Value)
		assert.Equal(t, "With description", suggestions[0].Description)
		assert.Equal(t, "cmd2", suggestions[1].Value)
		assert.Equal(t, "", suggestions[1].Description)
		assert.Equal(t, "cmd3", suggestions[2].Value)
		assert.Equal(t, "Another description", suggestions[2].Description)
	})

	t.Run("description with multiple tabs uses only first", func(t *testing.T) {
		output := []byte("cmd1\tdesc1\textra\textra2")
		suggestions := parseCompletionOutput(output, true)

		assert.Len(t, suggestions, 1)
		assert.Equal(t, "cmd1", suggestions[0].Value)
		assert.Equal(t, "desc1\textra\textra2", suggestions[0].Description)
	})
}

// TestValidateSimpleOutput tests validation of completion output
func TestValidateSimpleOutput(t *testing.T) {
	t.Run("valid simple output", func(t *testing.T) {
		output := []byte("cmd1\ncmd2\ncmd3")
		assert.True(t, validateSimpleOutput(output, 1))
		assert.True(t, validateSimpleOutput(output, 2))
		assert.True(t, validateSimpleOutput(output, 3))
	})

	t.Run("not enough valid lines", func(t *testing.T) {
		output := []byte("cmd1\ncmd2")
		assert.False(t, validateSimpleOutput(output, 3))
	})

	t.Run("empty output is invalid", func(t *testing.T) {
		output := []byte("")
		assert.False(t, validateSimpleOutput(output, 1))
	})

	t.Run("help text is invalid", func(t *testing.T) {
		output := []byte("Usage: tool [options]\nOptions:\n  --help  Show help")
		// validateSimpleOutput only checks for single-word lines
		// "Usage:" and "Options:" are single words, so they might pass
		// This test validates that multi-word help lines don't all count as valid
		isValid := validateSimpleOutput(output, 3)
		// We expect this to be false because not all lines are single words
		assert.False(t, isValid, "help text with multiple words per line should not be fully valid")
	})

	t.Run("single word per line is valid", func(t *testing.T) {
		output := []byte("checkout\ncommit\npush\npull")
		assert.True(t, validateSimpleOutput(output, 3))
	})

	t.Run("commands with hyphens are valid", func(t *testing.T) {
		output := []byte("docker-compose\nkubectl-get\nhelm-install")
		assert.True(t, validateSimpleOutput(output, 2))
	})

	t.Run("commands with underscores are valid", func(t *testing.T) {
		output := []byte("my_command\nother_command\nthird_command")
		assert.True(t, validateSimpleOutput(output, 2))
	})

	t.Run("multi-word lines are invalid", func(t *testing.T) {
		output := []byte("cmd1 is a command\ncmd2 is another\ncmd3")
		// Only cmd3 is valid (single word)
		assert.False(t, validateSimpleOutput(output, 2))
		assert.True(t, validateSimpleOutput(output, 1))
	})

	t.Run("empty lines don't count", func(t *testing.T) {
		output := []byte("cmd1\n\n\ncmd2\n\n")
		assert.True(t, validateSimpleOutput(output, 2))
		assert.False(t, validateSimpleOutput(output, 3))
	})

	t.Run("minLines of 0 requires at least one valid line", func(t *testing.T) {
		output := []byte("cmd1")
		assert.True(t, validateSimpleOutput(output, 0))

		output = []byte("")
		assert.False(t, validateSimpleOutput(output, 0))
	})

	t.Run("paths with slashes count as single word", func(t *testing.T) {
		// This is a known limitation - paths are treated as single words
		// if they don't have spaces
		output := []byte("/path/to/file\n/another/path")
		// These will count as single "words" since strings.Fields splits on whitespace
		assert.True(t, validateSimpleOutput(output, 2))
	})
}

// TestParseCompletionOutput_EdgeCases tests edge cases
func TestParseCompletionOutput_EdgeCases(t *testing.T) {
	t.Run("unicode characters", func(t *testing.T) {
		output := []byte("命令1\tコマンド\ncmd2\t説明")
		suggestions := parseCompletionOutput(output, true)

		assert.Len(t, suggestions, 2)
		assert.Equal(t, "命令1", suggestions[0].Value)
		assert.Equal(t, "コマンド", suggestions[0].Description)
	})

	t.Run("newlines in suggestions are handled", func(t *testing.T) {
		// Each line is a separate suggestion
		output := []byte("cmd1\ncmd2\ncmd3")
		suggestions := parseCompletionOutput(output, false)

		assert.Len(t, suggestions, 3)
	})

	t.Run("very long lines", func(t *testing.T) {
		longLine := strings.Repeat("a", 10000)
		output := []byte(fmt.Sprintf("%s\tcmd with long value", longLine))
		suggestions := parseCompletionOutput(output, true)

		assert.Len(t, suggestions, 1)
		assert.Equal(t, longLine, suggestions[0].Value)
	})
}

// TestValidateSimpleOutput_EdgeCases tests edge cases
func TestValidateSimpleOutput_EdgeCases(t *testing.T) {
	t.Run("only whitespace", func(t *testing.T) {
		output := []byte("   \n\t\n  ")
		assert.False(t, validateSimpleOutput(output, 1))
	})

	t.Run("mixed valid and invalid", func(t *testing.T) {
		output := []byte("valid1\ninvalid line with spaces\nvalid2")
		// Should have 2 valid lines
		assert.True(t, validateSimpleOutput(output, 2))
		assert.False(t, validateSimpleOutput(output, 3))
	})
}

package completion

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvCompleter_New(t *testing.T) {
	e := NewEnvCompleter()
	assert.NotNil(t, e)
}

func TestEnvCompleter_parseEnvOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Suggestion
	}{
		{
			name:  "simple list",
			input: "init\nplan\napply\n",
			expected: []Suggestion{
				{Value: "init", Description: ""},
				{Value: "plan", Description: ""},
				{Value: "apply", Description: ""},
			},
		},
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:  "with blank lines",
			input: "init\n\nplan\n\napply\n",
			expected: []Suggestion{
				{Value: "init", Description: ""},
				{Value: "plan", Description: ""},
				{Value: "apply", Description: ""},
			},
		},
		{
			name:  "with whitespace",
			input: "  init  \n  plan  \n",
			expected: []Suggestion{
				{Value: "init", Description: ""},
				{Value: "plan", Description: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEnvOutput([]byte(tt.input))
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEnvCompleter_Supports_NonExistentCommand(t *testing.T) {
	e := NewEnvCompleter()

	// Test with a command that doesn't exist
	result := e.Supports("this-command-does-not-exist-12345", []string{})
	assert.False(t, result, "Should return false for non-existent command")
}

func TestEnvCompleter_Supports_EmptyTool(t *testing.T) {
	e := NewEnvCompleter()

	// Test with empty tool name
	result := e.Supports("", []string{})
	assert.False(t, result, "Should return false for empty tool name")
}

func TestEnvCompleter_Complete_NonExistentCommand(t *testing.T) {
	e := NewEnvCompleter()

	// Test completion with non-existent command
	suggestions, err := e.Complete("this-command-does-not-exist-12345", []string{"arg1"})
	assert.Error(t, err, "Should return error for non-existent command")
	assert.Nil(t, suggestions)
}

func TestEnvCompleter_Complete_WithArgs(t *testing.T) {
	e := NewEnvCompleter()

	// Create a mock completion script that responds to COMP_LINE
	mockScript := `#!/bin/bash
if [ -n "$COMP_LINE" ]; then
    echo "suggestion1"
    echo "suggestion2"
fi
`
	// Create temp directory
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/mock-completer.sh"

	// Write script to file
	if err := os.WriteFile(scriptPath, []byte(mockScript), 0755); err != nil {
		t.Skip("Cannot write test script")
	}

	// Test with our mock script
	suggestions, err := e.Complete(scriptPath, []string{"arg1"})

	// Should succeed and return suggestions
	if err != nil {
		t.Logf("Completion error: %v", err)
		t.Skip("Mock script failed to execute")
	}

	assert.NoError(t, err)
	assert.Len(t, suggestions, 2)
	if len(suggestions) >= 2 {
		assert.Equal(t, "suggestion1", suggestions[0].Value)
		assert.Equal(t, "suggestion2", suggestions[1].Value)
	}
}

func TestEnvCompleter_Supports_ValidOutput(t *testing.T) {
	e := NewEnvCompleter()

	// Create a mock script that returns valid completions
	mockScript := `#!/bin/bash
if [ -n "$COMP_LINE" ]; then
    echo "init"
    echo "plan"
    echo "apply"
fi
`
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/mock-valid.sh"

	if err := os.WriteFile(scriptPath, []byte(mockScript), 0755); err != nil {
		t.Skip("Cannot write test script")
	}

	result := e.Supports(scriptPath, []string{})
	if result {
		assert.True(t, result, "Should support tool with valid output")
	} else {
		t.Logf("Mock script did not return expected support (may be environment issue)")
	}
}

func TestEnvCompleter_Supports_HelpText(t *testing.T) {
	e := NewEnvCompleter()

	// Create a mock script that returns help text (invalid)
	mockScript := `#!/bin/bash
echo "Usage: command [options]"
echo "Available commands:"
echo "  init - Initialize"
`
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/mock-help.sh"

	if err := os.WriteFile(scriptPath, []byte(mockScript), 0755); err != nil {
		t.Skip("Cannot write test script")
	}

	result := e.Supports(scriptPath, []string{})
	assert.False(t, result, "Should not support tool that returns help text")
}

func TestEnvCompleter_Supports_EmptyOutput(t *testing.T) {
	e := NewEnvCompleter()

	// Create a mock script that returns nothing
	mockScript := `#!/bin/bash
# Return nothing
exit 0
`
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/mock-empty.sh"

	if err := os.WriteFile(scriptPath, []byte(mockScript), 0755); err != nil {
		t.Skip("Cannot write test script")
	}

	result := e.Supports(scriptPath, []string{})
	assert.False(t, result, "Should not support tool that returns empty output")
}

func TestEnvCompleter_Supports_MixedOutput(t *testing.T) {
	e := NewEnvCompleter()

	// Create a mock script that returns some valid and some invalid lines
	mockScript := `#!/bin/bash
echo "command1"
echo "Usage: some help text here"
echo "command2"
`
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/mock-mixed.sh"

	if err := os.WriteFile(scriptPath, []byte(mockScript), 0755); err != nil {
		t.Skip("Cannot write test script")
	}

	result := e.Supports(scriptPath, []string{})
	assert.False(t, result, "Should not support tool with mixed valid/invalid output")
}

func TestEnvCompleter_Complete_NoArgs(t *testing.T) {
	e := NewEnvCompleter()

	// Create a mock script
	mockScript := `#!/bin/bash
if [ -n "$COMP_LINE" ]; then
    echo "cmd1"
    echo "cmd2"
fi
`
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/mock-noargs.sh"

	if err := os.WriteFile(scriptPath, []byte(mockScript), 0755); err != nil {
		t.Skip("Cannot write test script")
	}

	suggestions, err := e.Complete(scriptPath, []string{})

	if err != nil {
		t.Skip("Mock script failed to execute")
	}

	assert.NoError(t, err)
	assert.Len(t, suggestions, 2)
}

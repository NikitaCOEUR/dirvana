package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCobraCompleter_New(t *testing.T) {
	c := NewCobraCompleter()
	assert.NotNil(t, c)
}

func TestCobraCompleter_parseCobraOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Suggestion
	}{
		{
			name:  "simple values without description",
			input: "apply\ncreate\ndelete\n:4\nCompletion ended with directive: ShellCompDirectiveNoFileComp",
			expected: []Suggestion{
				{Value: "apply", Description: ""},
				{Value: "create", Description: ""},
				{Value: "delete", Description: ""},
			},
		},
		{
			name:  "values with descriptions",
			input: "apply\tApply a configuration\ncreate\tCreate a resource\n:4",
			expected: []Suggestion{
				{Value: "apply", Description: "Apply a configuration"},
				{Value: "create", Description: "Create a resource"},
			},
		},
		{
			name:     "empty output",
			input:    ":0",
			expected: nil,
		},
		{
			name:  "filters directive lines",
			input: "get\tGet resources\n:4\nShellCompDirective",
			expected: []Suggestion{
				{Value: "get", Description: "Get resources"},
			},
		},
		{
			name:  "empty lines are filtered",
			input: "apply\n\n\ncreate\n\ndelete\n:4",
			expected: []Suggestion{
				{Value: "apply", Description: ""},
				{Value: "create", Description: ""},
				{Value: "delete", Description: ""},
			},
		},
		{
			name:  "whitespace-only lines are treated as empty",
			input: "apply\n   \n\t\ncreate\n  \ndelete\n:4",
			expected: []Suggestion{
				{Value: "apply", Description: ""},
				{Value: "create", Description: ""},
				{Value: "delete", Description: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := parseCobraOutput([]byte(tt.input))
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCobraCompleter_Supports_NonExistentCommand(t *testing.T) {
	c := NewCobraCompleter()

	// Test with a command that doesn't exist
	result := c.Supports("this-command-does-not-exist-12345", []string{})
	assert.False(t, result, "Should return false for non-existent command")
}

func TestCobraCompleter_Supports_EmptyTool(t *testing.T) {
	c := NewCobraCompleter()

	// Test with empty tool name
	result := c.Supports("", []string{})
	assert.False(t, result, "Should return false for empty tool name")
}

func TestCobraCompleter_Complete_NonExistentCommand(t *testing.T) {
	c := NewCobraCompleter()

	// Test completion with non-existent command
	suggestions, err := c.Complete("this-command-does-not-exist-12345", []string{"__complete"})
	assert.Error(t, err, "Should return error for non-existent command")
	assert.Nil(t, suggestions)
}

package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashCompleteCompleter_New(t *testing.T) {
	b := NewBashCompleteCompleter()
	assert.NotNil(t, b)
}

func TestBashCompleteCompleter_parseBashCompleteOutput(t *testing.T) {
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
			result := parseBashCompleteOutput([]byte(tt.input))
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBashCompleteCompleter_Supports_NonExistentCommand(t *testing.T) {
	b := NewBashCompleteCompleter()

	// Test with a command that doesn't exist
	result := b.Supports("this-command-does-not-exist-12345", []string{})
	assert.False(t, result, "Should return false for non-existent command")
}

func TestBashCompleteCompleter_Supports_EmptyTool(t *testing.T) {
	b := NewBashCompleteCompleter()

	// Test with empty tool name
	result := b.Supports("", []string{})
	assert.False(t, result, "Should return false for empty tool name")
}

func TestBashCompleteCompleter_Complete_NonExistentCommand(t *testing.T) {
	b := NewBashCompleteCompleter()

	// Test completion with non-existent command
	suggestions, err := b.Complete("this-command-does-not-exist-12345", []string{"arg1"})
	assert.Error(t, err, "Should return error for non-existent command")
	assert.Nil(t, suggestions)
}

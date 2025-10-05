package completion

import (
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

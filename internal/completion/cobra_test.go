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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCobraOutput([]byte(tt.input))
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

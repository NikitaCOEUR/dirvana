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

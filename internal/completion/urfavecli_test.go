package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrfaveCliCompleter_New(t *testing.T) {
	u := NewUrfaveCliCompleter()
	assert.NotNil(t, u)
}

func TestUrfaveCliCompleter_parseUrfaveCliOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Suggestion
	}{
		{
			name:  "simple list",
			input: "export\nallow\nrevoke\nlist\n",
			expected: []Suggestion{
				{Value: "export", Description: ""},
				{Value: "allow", Description: ""},
				{Value: "revoke", Description: ""},
				{Value: "list", Description: ""},
			},
		},
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:  "with blank lines",
			input: "init\n\ninstall\n\nupdate\n",
			expected: []Suggestion{
				{Value: "init", Description: ""},
				{Value: "install", Description: ""},
				{Value: "update", Description: ""},
			},
		},
		{
			name:  "with whitespace",
			input: "  export  \n  allow  \n",
			expected: []Suggestion{
				{Value: "export", Description: ""},
				{Value: "allow", Description: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUrfaveCliOutput([]byte(tt.input))
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

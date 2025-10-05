package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuggestion(t *testing.T) {
	s := Suggestion{
		Value:       "test",
		Description: "test description",
	}

	assert.Equal(t, "test", s.Value)
	assert.Equal(t, "test description", s.Description)
}

func TestResult(t *testing.T) {
	result := Result{
		Suggestions: []Suggestion{
			{Value: "test1", Description: "desc1"},
			{Value: "test2", Description: "desc2"},
		},
		Source: "TestCompleter",
	}

	assert.Equal(t, 2, len(result.Suggestions))
	assert.Equal(t, "TestCompleter", result.Source)
}

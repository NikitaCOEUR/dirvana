package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlagCompleter_New(t *testing.T) {
	f := NewFlagCompleter()
	assert.NotNil(t, f)
}

func TestFlagCompleter_Supports_NonExistentCommand(t *testing.T) {
	f := NewFlagCompleter()

	// Test with a command that doesn't exist
	result := f.Supports("this-command-does-not-exist-12345", []string{})
	assert.False(t, result, "Should return false for non-existent command")
}

func TestFlagCompleter_Supports_EmptyTool(t *testing.T) {
	f := NewFlagCompleter()

	// Test with empty tool name
	result := f.Supports("", []string{})
	assert.False(t, result, "Should return false for empty tool name")
}

func TestFlagCompleter_Complete_NonExistentCommand(t *testing.T) {
	f := NewFlagCompleter()

	// Test completion with non-existent command
	suggestions, err := f.Complete("this-command-does-not-exist-12345", []string{"arg1"})
	assert.Error(t, err, "Should return error for non-existent command")
	assert.Nil(t, suggestions)
}

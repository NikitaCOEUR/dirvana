package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Filter(t *testing.T) {
	engine := &Engine{}

	suggestions := []Suggestion{
		{Value: "apply", Description: "Apply a configuration"},
		{Value: "annotate", Description: "Update annotations"},
		{Value: "get", Description: "Get resources"},
	}

	// Test with empty prefix (should return all)
	filtered := engine.Filter(suggestions, "")
	assert.Equal(t, 3, len(filtered))

	// Test with prefix "ap"
	filtered = engine.Filter(suggestions, "ap")
	assert.Equal(t, 1, len(filtered))
	assert.Equal(t, "apply", filtered[0].Value)

	// Test with prefix "a"
	filtered = engine.Filter(suggestions, "a")
	assert.Equal(t, 2, len(filtered))

	// Test with no matches
	filtered = engine.Filter(suggestions, "xyz")
	assert.Equal(t, 0, len(filtered))
}

func TestNewEngine(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(tmpDir)

	require.NotNil(t, engine)
	assert.Equal(t, 3, len(engine.completers)) // Cobra, UrfaveCli, BashComplete
	assert.NotNil(t, engine.detectionCache)
	assert.Equal(t, 3, len(engine.completerByName))
}

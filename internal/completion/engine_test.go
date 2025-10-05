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
	assert.Equal(t, 4, len(engine.completers)) // Cobra, Flag, Env, Script
	assert.NotNil(t, engine.detectionCache)
	assert.Equal(t, 4, len(engine.completerByName))
}

func TestEngine_Complete_NoCommand(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(tmpDir)

	result, err := engine.Complete("nonexistent-command", []string{"test"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Suggestions))
}

func TestEngine_Complete_EmptyWords(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(tmpDir)

	result, err := engine.Complete("echo", []string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Mock completer for testing
type mockCompleter struct {
	supportsResult bool
	suggestions    []Suggestion
	err            error
}

func (m *mockCompleter) Supports(_ string, _ []string) bool {
	return m.supportsResult
}

func (m *mockCompleter) Complete(_ string, _ []string) ([]Suggestion, error) {
	return m.suggestions, m.err
}

func TestEngine_Complete_WithMockCompleter(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(tmpDir)

	// Add a mock completer that supports the tool
	mock := &mockCompleter{
		supportsResult: true,
		suggestions: []Suggestion{
			{Value: "test1", Description: "Test 1"},
			{Value: "test2", Description: "Test 2"},
		},
		err: nil,
	}

	// Replace the completers with our mock
	engine.completers = []Completer{mock}
	engine.completerByName["Mock"] = mock

	// Test successful completion
	result, err := engine.Complete("mockTool", []string{"arg1"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Suggestions))
	assert.Equal(t, "test1", result.Suggestions[0].Value)
	assert.Equal(t, "test2", result.Suggestions[1].Value)
}

func TestEngine_Complete_WithCache(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(tmpDir)

	// Add a mock completer
	mock := &mockCompleter{
		supportsResult: true,
		suggestions: []Suggestion{
			{Value: "cached1", Description: "Cached 1"},
		},
		err: nil,
	}

	engine.completerByName["Mock"] = mock

	// Manually set cache entry
	engine.detectionCache.Set("cachedTool", "Mock")

	// Test completion with cached completer type
	result, err := engine.Complete("cachedTool", []string{"arg1"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Suggestions))
	assert.Contains(t, result.Source, "cached")
}

func TestEngine_Complete_CacheInvalidAfterFailure(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(tmpDir)

	// Add a failing mock completer
	failingMock := &mockCompleter{
		supportsResult: true,
		suggestions:    nil,
		err:            assert.AnError,
	}

	// Add a successful mock completer
	successMock := &mockCompleter{
		supportsResult: true,
		suggestions: []Suggestion{
			{Value: "success", Description: "Success"},
		},
		err: nil,
	}

	engine.completers = []Completer{failingMock, successMock}
	engine.completerByName["FailingMock"] = failingMock

	// Set cache to failing completer
	engine.detectionCache.Set("tool", "FailingMock")

	// Should fallback to trying other completers
	result, err := engine.Complete("tool", []string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Should find successMock
	assert.Equal(t, 1, len(result.Suggestions))
	assert.Equal(t, "success", result.Suggestions[0].Value)
}

func TestEngine_Complete_NoSuggestions(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewEngine(tmpDir)

	// Add a mock that supports but returns no suggestions
	mock := &mockCompleter{
		supportsResult: true,
		suggestions:    []Suggestion{},
		err:            nil,
	}

	engine.completers = []Completer{mock}

	result, err := engine.Complete("tool", []string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Suggestions))
	assert.Equal(t, "none", result.Source)
}

func TestEngine_getCompleterType(t *testing.T) {
	cobra := NewCobraCompleter()
	flag := NewFlagCompleter()
	env := NewEnvCompleter()
	script := NewScriptCompleter()

	assert.Equal(t, "Cobra", getCompleterType(cobra))
	assert.Equal(t, "Flag", getCompleterType(flag))
	assert.Equal(t, "Env", getCompleterType(env))
	assert.Equal(t, "Script", getCompleterType(script))
}

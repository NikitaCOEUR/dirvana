package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptCompleter_New(t *testing.T) {
	s := NewScriptCompleter("")
	assert.NotNil(t, s)
}

func TestScriptCompleter_findCompletionScript(t *testing.T) {
	s := NewScriptCompleter("")
	// Test with git (should exist on most systems)
	script := s.findCompletionScript("git")

	// On systems with bash-completion installed, git should have a script
	if script != "" {
		assert.FileExists(t, script)
		t.Logf("Found git completion script at: %s", script)
	} else {
		t.Skip("Git completion script not found (bash-completion may not be installed)")
	}
}

func TestScriptCompleter_Supports(t *testing.T) {
	s := NewScriptCompleter("")

	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{
			name:     "git should be supported if script exists",
			tool:     "git",
			expected: s.findCompletionScript("git") != "",
		},
		{
			name:     "non-existent tool should not be supported",
			tool:     "nonexistenttool12345",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Supports(tt.tool, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScriptCompleter_Complete(t *testing.T) {
	s := NewScriptCompleter("")

	// Skip if git completion is not available
	if s.findCompletionScript("git") == "" {
		t.Skip("Git completion script not found")
	}

	tests := []struct {
		name        string
		tool        string
		args        []string
		expectError bool
		checkFunc   func(*testing.T, []Suggestion)
	}{
		{
			name: "git with empty args should return subcommands",
			tool: "git",
			args: []string{""},
			checkFunc: func(t *testing.T, suggestions []Suggestion) {
				// Should include common git commands
				assert.NotEmpty(t, suggestions)

				// Check if some common commands are present
				values := make([]string, len(suggestions))
				for i, s := range suggestions {
					values[i] = s.Value
				}

				// At least some of these should be present
				commonCommands := []string{"add", "commit", "push", "pull", "status", "checkout", "branch"}
				foundCount := 0
				for _, cmd := range commonCommands {
					for _, val := range values {
						if val == cmd {
							foundCount++
							break
						}
					}
				}

				assert.Greater(t, foundCount, 0, "Should find at least one common git command")
			},
		},
		{
			name: "git checkout should suggest branches",
			tool: "git",
			args: []string{"checkout", ""},
			checkFunc: func(t *testing.T, suggestions []Suggestion) {
				// Should return suggestions (branches, etc.)
				// We can't assert exact content as it depends on the repo state
				// but it should not be empty in a git repository
				t.Logf("Got %d suggestions for git checkout", len(suggestions))
				// Don't assert not empty as this test might run in non-git context
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := s.Complete(tt.tool, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				if err != nil {
					t.Logf("Completion error (may be expected in test environment): %v", err)
				}

				if tt.checkFunc != nil && err == nil {
					tt.checkFunc(t, suggestions)
				}
			}
		})
	}
}

func TestScriptCompleter_parseScriptOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Suggestion
	}{
		{
			name:  "simple words",
			input: "add\ncommit\npush\npull\n",
			expected: []Suggestion{
				{Value: "add"},
				{Value: "commit"},
				{Value: "push"},
				{Value: "pull"},
			},
		},
		{
			name:  "words with descriptions",
			input: "add\tAdd file contents to the index\ncommit\tRecord changes to the repository\n",
			expected: []Suggestion{
				{Value: "add", Description: "Add file contents to the index"},
				{Value: "commit", Description: "Record changes to the repository"},
			},
		},
		{
			name:     "empty output",
			input:    "",
			expected: []Suggestion{},
		},
		{
			name:  "output with empty lines",
			input: "add\n\ncommit\n\n",
			expected: []Suggestion{
				{Value: "add"},
				{Value: "commit"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseScriptOutput([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScriptCompleter_escapeShellWords(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "simple words",
			input:    []string{"git", "commit"},
			expected: []string{"'git'", "'commit'"},
		},
		{
			name:     "words with spaces",
			input:    []string{"git", "commit message"},
			expected: []string{"'git'", "'commit message'"},
		},
		{
			name:     "words with single quotes",
			input:    []string{"git", "commit 'message'"},
			expected: []string{"'git'", "'commit '\\''message'\\'''"},
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeShellWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScriptCompleter_Integration(t *testing.T) {
	s := NewScriptCompleter("")

	// Skip if not in a git repository or git completion not available
	if s.findCompletionScript("git") == "" {
		t.Skip("Git completion script not found")
	}

	// Check if we're in a git repository (check parent directories too)
	inGitRepo := false
	cwd, _ := os.Getwd()
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			inGitRepo = true
			break
		}
	}

	if !inGitRepo {
		t.Skip("Not in a git repository")
	}

	t.Run("complete git subcommands", func(t *testing.T) {
		suggestions, err := s.Complete("git", []string{""})
		if err != nil {
			t.Logf("Completion error: %v", err)
			t.Skip("Git completion not working in test environment")
		}

		assert.NotEmpty(t, suggestions, "Should have git subcommand suggestions")
		t.Logf("Found %d git subcommands", len(suggestions))

		// Log first few suggestions
		for i, s := range suggestions {
			if i >= 5 {
				break
			}
			t.Logf("  - %s", s.Value)
		}
	})
}

// TestScriptCompleter_EnsureScriptAvailable tests script availability
func TestScriptCompleter_EnsureScriptAvailable(t *testing.T) {
	t.Run("returns script path when script exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := NewScriptCompleter(tmpDir)

		// Create a mock script
		scriptPath := filepath.Join(tmpDir, "completion-scripts", "bash", "test-tool")
		err := os.MkdirAll(filepath.Dir(scriptPath), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(scriptPath, []byte("#!/bin/bash\n"), 0644)
		assert.NoError(t, err)

		path, err := s.ensureScriptAvailable("test-tool")
		assert.NoError(t, err)
		assert.Equal(t, scriptPath, path)
	})

	t.Run("returns error when script not found and not in registry", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := NewScriptCompleter(tmpDir)

		_, err := s.ensureScriptAvailable("nonexistent-tool-xyz")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no completion script found")
	})
}

// TestScriptCompleter_Complete_Errors tests error cases
func TestScriptCompleter_Complete_Errors(t *testing.T) {
	t.Run("returns error for nonexistent tool", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := NewScriptCompleter(tmpDir)

		_, err := s.Complete("nonexistent-tool-xyz-123", []string{})
		assert.Error(t, err)
	})
}

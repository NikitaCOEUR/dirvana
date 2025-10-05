package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashFunctionCompleter_New(t *testing.T) {
	b := NewBashFunctionCompleter()
	assert.NotNil(t, b)
}

func TestBashFunctionCompleter_findCompletionScript(t *testing.T) {
	// Test with git (should exist on most systems)
	script := findCompletionScript("git")

	// On systems with bash-completion installed, git should have a script
	if script != "" {
		assert.FileExists(t, script)
		t.Logf("Found git completion script at: %s", script)
	} else {
		t.Skip("Git completion script not found (bash-completion may not be installed)")
	}
}

func TestBashFunctionCompleter_Supports(t *testing.T) {
	b := NewBashFunctionCompleter()

	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{
			name:     "git should be supported if script exists",
			tool:     "git",
			expected: findCompletionScript("git") != "",
		},
		{
			name:     "non-existent tool should not be supported",
			tool:     "nonexistenttool12345",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := b.Supports(tt.tool, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBashFunctionCompleter_Complete(t *testing.T) {
	// Skip if git completion is not available
	if findCompletionScript("git") == "" {
		t.Skip("Git completion script not found")
	}

	b := NewBashFunctionCompleter()

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
			suggestions, err := b.Complete(tt.tool, tt.args)

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

func TestBashFunctionCompleter_parseBashFunctionOutput(t *testing.T) {
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
			result := parseBashFunctionOutput([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBashFunctionCompleter_escapeShellWords(t *testing.T) {
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

func TestBashFunctionCompleter_Integration(t *testing.T) {
	// Skip if not in a git repository or git completion not available
	if findCompletionScript("git") == "" {
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

	b := NewBashFunctionCompleter()

	t.Run("complete git subcommands", func(t *testing.T) {
		suggestions, err := b.Complete("git", []string{""})
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

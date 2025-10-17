package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name:  "empty lines are filtered",
			input: "apply\n\n\ncreate\n\ndelete\n:4",
			expected: []Suggestion{
				{Value: "apply", Description: ""},
				{Value: "create", Description: ""},
				{Value: "delete", Description: ""},
			},
		},
		{
			name:  "whitespace-only lines are treated as empty",
			input: "apply\n   \n\t\ncreate\n  \ndelete\n:4",
			expected: []Suggestion{
				{Value: "apply", Description: ""},
				{Value: "create", Description: ""},
				{Value: "delete", Description: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := parseCobraOutput([]byte(tt.input))
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCobraCompleter_Supports_NonExistentCommand(t *testing.T) {
	c := NewCobraCompleter()

	// Test with a command that doesn't exist
	result := c.Supports("this-command-does-not-exist-12345", []string{})
	assert.False(t, result, "Should return false for non-existent command")
}

func TestCobraCompleter_Supports_EmptyTool(t *testing.T) {
	c := NewCobraCompleter()

	// Test with empty tool name
	result := c.Supports("", []string{})
	assert.False(t, result, "Should return false for empty tool name")
}

func TestCobraCompleter_Complete_NonExistentCommand(t *testing.T) {
	c := NewCobraCompleter()

	// Test completion with non-existent command
	suggestions, err := c.Complete("this-command-does-not-exist-12345", []string{"__complete"})
	assert.Error(t, err, "Should return error for non-existent command")
	assert.Nil(t, suggestions)
}

func TestCobraCompleter_Complete_WithDirectives(t *testing.T) {
	// This test cannot easily mock a Cobra command, but we can test
	// the directive handling by directly testing completeFilesWithExtensions
	// and completeDirectories which are called based on directives
	c := NewCobraCompleter()

	t.Run("completeDirectories with empty args", func(t *testing.T) {
		// Test with no args - should use current directory
		suggestions, err := c.completeDirectories([]string{})
		assert.NoError(t, err)
		// Should return a slice (may be empty if no directories in test directory)
		assert.IsType(t, []Suggestion{}, suggestions)
	})

	t.Run("completeDirectories with directory prefix", func(t *testing.T) {
		// Test with a prefix
		suggestions, err := c.completeDirectories([]string{"d"})
		assert.NoError(t, err)
		assert.IsType(t, []Suggestion{}, suggestions)
	})

	t.Run("completeFilesWithExtensions with empty extensions", func(t *testing.T) {
		// Test with no extension suggestions
		suggestions, err := c.completeFilesWithExtensions([]Suggestion{}, []string{})
		assert.NoError(t, err)
		assert.IsType(t, []Suggestion{}, suggestions)
	})

	t.Run("completeFilesWithExtensions with args containing slash", func(t *testing.T) {
		// Test with path that ends with /
		suggestions, err := c.completeFilesWithExtensions(
			[]Suggestion{{Value: "go", Description: ""}},
			[]string{"./"},
		)
		assert.NoError(t, err)
		assert.IsType(t, []Suggestion{}, suggestions)
	})
}

func TestCobraCompleter_Complete_DirectiveHandling(t *testing.T) {
	tmpDir := t.TempDir()
	c := NewCobraCompleter()

	// Create a mock script that simulates Cobra __complete command
	// This will test the directive handling logic in Complete()
	t.Run("FilterFileExt directive (8)", func(t *testing.T) {
		// Create a script that returns file extensions with directive 8
		mockScript := `#!/bin/bash
echo "json"
echo "yaml"
echo ":8"
`
		scriptPath := filepath.Join(tmpDir, "mock-cobra-ext")
		require.NoError(t, os.WriteFile(scriptPath, []byte(mockScript), 0755))

		// Create some test files
		testDir := filepath.Join(tmpDir, "testfiles")
		require.NoError(t, os.MkdirAll(testDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "file.json"), []byte(""), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "file.yaml"), []byte(""), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "file.txt"), []byte(""), 0644))

		// Change to test directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()
		require.NoError(t, os.Chdir(testDir))

		// Call Complete which should trigger completeFilesWithExtensions
		// Pass empty args so it searches without prefix filter
		suggestions, err := c.Complete(scriptPath, []string{})
		assert.NoError(t, err)

		// Should return files with matching extensions
		values := make([]string, len(suggestions))
		for i, s := range suggestions {
			values[i] = s.Value
		}

		// Should include json and yaml files, not txt
		assert.Contains(t, values, "file.json")
		assert.Contains(t, values, "file.yaml")
	})

	t.Run("FilterDirs directive (16)", func(t *testing.T) {
		// Create a script that returns directive 16 (directories only)
		mockScript := `#!/bin/bash
echo ":16"
`
		scriptPath := filepath.Join(tmpDir, "mock-cobra-dirs")
		require.NoError(t, os.WriteFile(scriptPath, []byte(mockScript), 0755))

		// Create test structure with dirs and files
		testDir := filepath.Join(tmpDir, "testdirs")
		require.NoError(t, os.MkdirAll(testDir, 0755))
		require.NoError(t, os.Mkdir(filepath.Join(testDir, "dir1"), 0755))
		require.NoError(t, os.Mkdir(filepath.Join(testDir, "dir2"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "file.txt"), []byte(""), 0644))

		// Change to test directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()
		require.NoError(t, os.Chdir(testDir))

		// Call Complete which should trigger completeDirectories
		// Pass empty args so it searches without prefix filter
		suggestions, err := c.Complete(scriptPath, []string{})
		assert.NoError(t, err)

		// Should return only directories
		values := make([]string, len(suggestions))
		for i, s := range suggestions {
			values[i] = s.Value
		}
		assert.Contains(t, values, "dir1/")
		assert.Contains(t, values, "dir2/")
		// Should NOT contain the file
		assert.NotContains(t, values, "file.txt")
	})

	t.Run("No directive (0) - returns suggestions as-is", func(t *testing.T) {
		// Create a script that returns regular suggestions without special directives
		mockScript := `#!/bin/bash
echo "apply"
echo "create"
echo "delete"
echo ":0"
`
		scriptPath := filepath.Join(tmpDir, "mock-cobra-normal")
		require.NoError(t, os.WriteFile(scriptPath, []byte(mockScript), 0755))

		// Call Complete which should return suggestions as-is
		suggestions, err := c.Complete(scriptPath, []string{})
		assert.NoError(t, err)

		// Should return the suggestions without modification
		assert.Len(t, suggestions, 3)
		values := make([]string, len(suggestions))
		for i, s := range suggestions {
			values[i] = s.Value
		}
		assert.Contains(t, values, "apply")
		assert.Contains(t, values, "create")
		assert.Contains(t, values, "delete")
	})
}

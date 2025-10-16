package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCobraOutput_WithDirectives(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedSuggestions []Suggestion
		expectedDirective   int
	}{
		{
			name:  "directive FilterFileExt (8)",
			input: "json\nyaml\nyml\n:8",
			expectedSuggestions: []Suggestion{
				{Value: "json", Description: ""},
				{Value: "yaml", Description: ""},
				{Value: "yml", Description: ""},
			},
			expectedDirective: 8,
		},
		{
			name:  "directive FilterDirs (16)",
			input: ":16",
			expectedSuggestions: []Suggestion{},
			expectedDirective:   16,
		},
		{
			name:  "directive NoSpace (2)",
			input: "pods\nservices\n:2",
			expectedSuggestions: []Suggestion{
				{Value: "pods", Description: ""},
				{Value: "services", Description: ""},
			},
			expectedDirective: 2,
		},
		{
			name:  "directive NoFileComp (4)",
			input: "get\ndelete\n:4",
			expectedSuggestions: []Suggestion{
				{Value: "get", Description: ""},
				{Value: "delete", Description: ""},
			},
			expectedDirective: 4,
		},
		{
			name:  "combined directives (8+4=12)",
			input: "json\nyaml\n:12",
			expectedSuggestions: []Suggestion{
				{Value: "json", Description: ""},
				{Value: "yaml", Description: ""},
			},
			expectedDirective: 12,
		},
		{
			name:  "no directive (0)",
			input: "apply\ncreate\n:0",
			expectedSuggestions: []Suggestion{
				{Value: "apply", Description: ""},
				{Value: "create", Description: ""},
			},
			expectedDirective: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, directive := parseCobraOutput([]byte(tt.input))
			assert.Equal(t, tt.expectedSuggestions, suggestions)
			assert.Equal(t, tt.expectedDirective, directive)
		})
	}
}

func TestCobraCompleter_CompleteFilesWithExtensions(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.yaml"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.yml"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(""), 0644))

	// Create subdirectory
	subdir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subdir, 0755))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	c := NewCobraCompleter()

	// Test with extension suggestions
	extensionSuggestions := []Suggestion{
		{Value: "json", Description: ""},
		{Value: "yaml", Description: ""},
		{Value: "yml", Description: ""},
	}

	suggestions, err := c.completeFilesWithExtensions(extensionSuggestions, []string{})

	assert.NoError(t, err)
	assert.NotEmpty(t, suggestions)

	// Should include all matching files
	values := make([]string, len(suggestions))
	for i, s := range suggestions {
		values[i] = s.Value
	}

	assert.Contains(t, values, "file.json")
	assert.Contains(t, values, "file.yaml")
	assert.Contains(t, values, "file.yml")
	assert.NotContains(t, values, "file.txt", "should not include .txt files")
	assert.NotContains(t, values, "README.md", "should not include .md files")

	// Should include directory with trailing slash
	assert.Contains(t, values, "subdir/")
}

func TestCobraCompleter_CompleteDirectories(t *testing.T) {
	// Create temp directory with test structure
	tmpDir := t.TempDir()

	// Create directories
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dir2"), 0755))

	// Create files (should be excluded)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(""), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	c := NewCobraCompleter()
	suggestions, err := c.completeDirectories([]string{})

	assert.NoError(t, err)
	assert.NotEmpty(t, suggestions)

	// Should only include directories
	values := make([]string, len(suggestions))
	for i, s := range suggestions {
		values[i] = s.Value
	}

	assert.Contains(t, values, "dir1/")
	assert.Contains(t, values, "dir2/")
	assert.NotContains(t, values, "file.txt", "should not include files")
}

func TestCobraCompleter_CompleteFilesWithPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with common prefix
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test1.json"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test2.json"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other.json"), []byte(""), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	c := NewCobraCompleter()
	extensionSuggestions := []Suggestion{{Value: "json", Description: ""}}

	// Complete with prefix "test"
	suggestions, err := c.completeFilesWithExtensions(extensionSuggestions, []string{"test"})

	assert.NoError(t, err)

	values := make([]string, len(suggestions))
	for i, s := range suggestions {
		values[i] = s.Value
	}

	// Should match files starting with "test"
	assert.Contains(t, values, "test1.json")
	assert.Contains(t, values, "test2.json")
	assert.NotContains(t, values, "other.json", "should not match files without prefix")
}

func TestCobraCompleter_SkipsHiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden and visible files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hidden.json"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "visible.json"), []byte(""), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".hidden_dir"), 0755))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	c := NewCobraCompleter()
	extensionSuggestions := []Suggestion{{Value: "json", Description: ""}}

	suggestions, err := c.completeFilesWithExtensions(extensionSuggestions, []string{})

	assert.NoError(t, err)

	values := make([]string, len(suggestions))
	for i, s := range suggestions {
		values[i] = s.Value
	}

	assert.Contains(t, values, "visible.json")
	assert.NotContains(t, values, ".hidden.json", "should skip hidden files")
	assert.NotContains(t, values, ".hidden_dir/", "should skip hidden directories")
}

func TestCobraDirectiveConstants(t *testing.T) {
	// Verify directive constants match Cobra's values
	assert.Equal(t, 1, ShellCompDirectiveError)
	assert.Equal(t, 2, ShellCompDirectiveNoSpace)
	assert.Equal(t, 4, ShellCompDirectiveNoFileComp)
	assert.Equal(t, 8, ShellCompDirectiveFilterFileExt)
	assert.Equal(t, 16, ShellCompDirectiveFilterDirs)
	assert.Equal(t, 32, ShellCompDirectiveKeepOrder)
}

func TestCobraCompleter_HandlesInvalidDirectory(t *testing.T) {
	c := NewCobraCompleter()
	extensionSuggestions := []Suggestion{{Value: "json", Description: ""}}

	// Try to complete in non-existent directory
	suggestions, err := c.completeFilesWithExtensions(extensionSuggestions, []string{"/non/existent/path/file"})

	// Should return empty, not error (graceful degradation)
	assert.NoError(t, err)
	assert.Empty(t, suggestions)
}

func TestCobraCompleter_NavigatesSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectory with files
	subdir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "nested.json"), []byte(""), 0644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	c := NewCobraCompleter()
	extensionSuggestions := []Suggestion{{Value: "json", Description: ""}}

	// Complete with subdirectory path
	suggestions, err := c.completeFilesWithExtensions(extensionSuggestions, []string{"subdir/"})

	assert.NoError(t, err)
	assert.NotEmpty(t, suggestions)

	values := make([]string, len(suggestions))
	for i, s := range suggestions {
		values[i] = s.Value
	}

	// Should find file in subdirectory
	assert.Contains(t, values, "subdir/nested.json")
}

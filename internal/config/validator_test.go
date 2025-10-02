package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `aliases:
  ll: ls -la
  gs: git status
functions:
  greet: echo "Hello, $1!"
env:
  PROJECT_NAME: myproject
  GIT_BRANCH:
    sh: git branch --show-current
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 0)
}

func TestValidate_FileNotFound(t *testing.T) {
	_, err := Validate("/nonexistent/path/.dirvana.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file not found")
}

func TestValidate_InvalidSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `aliases:
  ll: ls -la
  invalid yaml here [[[
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Errors), 0)
	assert.Equal(t, "syntax", result.Errors[0].Field)
}

func TestValidate_NameConflict(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `aliases:
  test: echo alias
functions:
  test: echo function
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Message, "Name conflict")
	assert.Contains(t, result.Errors[0].Message, "test")
}

func TestValidate_EmptyAlias(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `aliases:
  ll: ls -la
  empty: ""
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Message, "Alias command is empty")
}

func TestValidate_EmptyFunction(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `functions:
  greet: echo hello
  empty: ""
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Message, "Function body is empty")
}

func TestValidate_EmptyShellCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `env:
  VAR1: static
  VAR2:
    sh: ""
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Message, "Shell command is empty")
}

func TestValidate_MultilineShellCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `env:
  VAR:
    sh: "line1\nline2"
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0].Message, "multiline")
}

func TestValidate_MultipleErrors(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `aliases:
  test: echo test
  empty: ""
functions:
  test: echo function
env:
  VAR:
    sh: ""
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	result, err := Validate(configPath)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	// Should have at least 3 errors: name conflict, empty alias, empty shell command
	assert.GreaterOrEqual(t, len(result.Errors), 3)
}

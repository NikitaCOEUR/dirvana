package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWithSchema_ValidYAML(t *testing.T) {
	content := []byte(`
aliases:
  ll: ls -lah
  test: echo test
functions:
  greet: echo "Hello $1"
env:
  PROJECT_NAME: myproject
  GIT_BRANCH:
    sh: git branch --show-current
local_only: false
ignore_global: false
`)

	result, err := ValidateWithSchema("test.yml", content)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateWithSchema_InvalidAliasName(t *testing.T) {
	content := []byte(`
aliases:
  123invalid: bad command
`)

	result, err := ValidateWithSchema("test.yml", content)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "123invalid")
}

func TestValidateWithSchema_InvalidEnvVarName(t *testing.T) {
	content := []byte(`
env:
  123BAD: value
`)

	result, err := ValidateWithSchema("test.yml", content)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestValidateWithSchema_InvalidEnvVarShell(t *testing.T) {
	content := []byte(`
env:
  VAR:
    sh: ""
`)

	result, err := ValidateWithSchema("test.yml", content)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestValidateWithSchema_ValidYAMLExtension(t *testing.T) {
	content := []byte(`
aliases:
  ll: ls -lah
`)

	result, err := ValidateWithSchema("test.yaml", content)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateWithSchema_ValidJSON(t *testing.T) {
	content := []byte(`{
  "aliases": {
    "ll": "ls -lah"
  },
  "env": {
    "VAR": "value"
  }
}`)

	result, err := ValidateWithSchema("test.json", content)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateWithSchema_InvalidJSONSyntax(t *testing.T) {
	content := []byte(`{invalid json`)

	result, err := ValidateWithSchema("test.json", content)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0].Field, "syntax")
}

func TestValidateWithSchema_InvalidYAMLSyntax(t *testing.T) {
	content := []byte(`
aliases:
  - invalid yaml structure
`)

	result, err := ValidateWithSchema("test.yml", content)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	// This YAML is valid syntax but wrong schema - aliases should be an object not array
	assert.NotEmpty(t, result.Errors)
}

func TestValidateWithSchema_ValidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	tomlFile := filepath.Join(tmpDir, "test.toml")

	content := []byte(`
[aliases]
ll = "ls -lah"

[env]
PROJECT = "test"
`)
	require.NoError(t, os.WriteFile(tomlFile, content, 0644))

	result, err := ValidateWithSchema(tomlFile, content)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateWithSchema_InvalidTOML(t *testing.T) {
	content := []byte(`[invalid toml syntax`)

	result, err := ValidateWithSchema("test.toml", content)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "TOML")
}

func TestValidateWithSchema_UnsupportedFormat(t *testing.T) {
	content := []byte(`some content`)

	_, err := ValidateWithSchema("test.txt", content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file format")
}

func TestValidateWithSchema_InvalidYAMLSchema(t *testing.T) {
	content := []byte(`
aliases:
  - invalid: structure
`)

	result, err := ValidateWithSchema("test.yml", content)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestGetSchemaJSON(t *testing.T) {
	schema := GetSchemaJSON()
	assert.NotEmpty(t, schema)
	assert.Contains(t, schema, "draft-07")
	assert.Contains(t, schema, "Dirvana Configuration")
	assert.Contains(t, schema, "aliases")
	assert.Contains(t, schema, "functions")
	assert.Contains(t, schema, "env")
}

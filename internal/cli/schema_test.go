package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_PrintToStdout(t *testing.T) {
	// Capture stdout is not straightforward in Go tests
	// Instead we test that Schema with empty path doesn't error
	err := Schema("")
	require.NoError(t, err)
}

func TestSchema_WriteToFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test-schema.json")

	// Write schema to file
	err := Schema(outputFile)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err, "Schema file should be created")

	// Verify file contains valid JSON schema
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	schemaStr := string(content)
	assert.Contains(t, schemaStr, `"$schema": "http://json-schema.org/draft-07/schema#"`)
	assert.Contains(t, schemaStr, `"title": "Dirvana Configuration"`)
	assert.Contains(t, schemaStr, `"aliases"`)
	assert.Contains(t, schemaStr, `"functions"`)
	assert.Contains(t, schemaStr, `"env"`)
	assert.Contains(t, schemaStr, `"local_only"`)
	assert.Contains(t, schemaStr, `"ignore_global"`)
}

func TestSchema_WriteToFile_InvalidPath(t *testing.T) {
	// Try to write to invalid path
	err := Schema("/nonexistent/directory/schema.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write schema")
}

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validAliasConfig = `aliases:
  ll: ls -la
`

func TestValidate_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `aliases:
  ll: ls -la
functions:
  greet: echo hello
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := Validate(configPath)
	require.NoError(t, err)
}

func TestValidate_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	content := `aliases:
  test: echo test
  empty: ""
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := Validate(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidate_AutoDetect(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create config
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(validAliasConfig), 0644))

	// Should auto-detect config in current dir
	err = Validate("")
	require.NoError(t, err)
}

func TestValidate_NoConfigFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// No config file
	err = Validate("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no config file found")
}

func TestValidate_FileNotExist(t *testing.T) {
	err := Validate("/nonexistent/path/.dirvana.yml")
	require.Error(t, err)
}

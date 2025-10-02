package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/stretchr/testify/require"
)

func TestEdit_CreatesConfigIfNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Set a valid editor that just exits (for testing)
	t.Setenv("EDITOR", "true")

	// Run edit - should create config
	err = Edit()
	require.NoError(t, err)

	// Verify config was created
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Verify content is valid
	loader := config.New()
	cfg, err := loader.Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestEdit_OpensExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create existing config
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte(testAliasConfig), 0644))

	// Set a valid editor that just exits (for testing)
	t.Setenv("EDITOR", "true")

	// Run edit - should open existing config
	err = Edit()
	require.NoError(t, err)

	// Verify config still exists and wasn't overwritten
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "test: echo test")
}

func TestEdit_NoEditorAvailable(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Unset all editor vars and use empty PATH
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	t.Setenv("PATH", "")

	// Run edit - should fail with no editor
	err = Edit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no editor found")
}

package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDetectionCacheInfo_NoCache tests when detection cache doesn't exist
func TestGetDetectionCacheInfo_NoCache(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := GetDetectionCacheInfo(tmpDir)
	require.NoError(t, err)
	assert.Nil(t, result) // Should return nil when no cache exists
}

// TestGetDetectionCacheInfo_InvalidJSON tests with invalid JSON
func TestGetDetectionCacheInfo_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	detectionPath := filepath.Join(tmpDir, "completion-detection.json")

	// Write invalid JSON
	err := os.WriteFile(detectionPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	result, err := GetDetectionCacheInfo(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return partial info (path and size, but no commands)
	assert.Equal(t, detectionPath, result.Path)
	assert.Greater(t, result.Size, int64(0))
	assert.Empty(t, result.Commands)
}

// TestGetDetectionCacheInfo_EmptyJSON tests with empty JSON object
func TestGetDetectionCacheInfo_EmptyJSON(t *testing.T) {
	tmpDir := t.TempDir()
	detectionPath := filepath.Join(tmpDir, "completion-detection.json")

	err := os.WriteFile(detectionPath, []byte("{}"), 0644)
	require.NoError(t, err)

	result, err := GetDetectionCacheInfo(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, detectionPath, result.Path)
	assert.Empty(t, result.Commands)
}

// TestGetRegistryInfo_NoRegistry tests when registry doesn't exist
func TestGetRegistryInfo_NoRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := GetRegistryInfo(tmpDir)
	require.NoError(t, err)
	assert.Nil(t, result) // Should return nil when no registry exists
}

// TestGetRegistryInfo_InvalidYAML tests with invalid YAML
func TestGetRegistryInfo_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "completion-registry-v1.yml")

	// Write invalid YAML
	err := os.WriteFile(registryPath, []byte("not: valid: yaml: structure"), 0644)
	require.NoError(t, err)

	result, err := GetRegistryInfo(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return partial info
	assert.Equal(t, registryPath, result.Path)
	assert.Greater(t, result.Size, int64(0))
	assert.Equal(t, 0, result.ToolsCount) // Invalid YAML means no tools parsed
}

// TestGetRegistryInfo_EmptyTools tests with empty tools section
func TestGetRegistryInfo_EmptyTools(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "completion-registry-v1.yml")

	err := os.WriteFile(registryPath, []byte("tools: {}"), 0644)
	require.NoError(t, err)

	result, err := GetRegistryInfo(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.ToolsCount)
}

// TestGetDownloadedScripts_NoScripts tests when scripts directory doesn't exist
func TestGetDownloadedScripts_NoScripts(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := GetDownloadedScripts(tmpDir)
	require.NoError(t, err)
	assert.Nil(t, result) // Should return nil when no scripts exist
}

// TestGetDownloadedScripts_EmptyDirectory tests with empty scripts directory
func TestGetDownloadedScripts_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsPath := filepath.Join(tmpDir, "completion-scripts", "bash")
	err := os.MkdirAll(scriptsPath, 0755)
	require.NoError(t, err)

	result, err := GetDownloadedScripts(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, result) // Should return empty slice, not nil
}

// TestGetDownloadedScripts_WithSubdirectories tests that subdirectories are skipped
func TestGetDownloadedScripts_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsPath := filepath.Join(tmpDir, "completion-scripts", "bash")
	err := os.MkdirAll(scriptsPath, 0755)
	require.NoError(t, err)

	// Create a subdirectory (should be skipped)
	err = os.MkdirAll(filepath.Join(scriptsPath, "subdir"), 0755)
	require.NoError(t, err)

	// Create a file
	err = os.WriteFile(filepath.Join(scriptsPath, "tool.sh"), []byte("#!/bin/bash"), 0755)
	require.NoError(t, err)

	result, err := GetDownloadedScripts(tmpDir)
	require.NoError(t, err)

	// Should only have the file, not the subdirectory
	assert.Len(t, result, 1)
	assert.Equal(t, "tool.sh", result[0].Tool)
}

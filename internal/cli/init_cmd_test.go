package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleConfigMarkers are the sections every generated config must carry
var sampleConfigMarkers = []string{
	"yaml-language-server: $schema=",
	"aliases:",
	"functions:",
	"env:",
	"local_only:",
}

func TestInit(t *testing.T) {
	env := newTestEnv(t)

	out := captureStdout(t, func() {
		require.NoError(t, Init(false))
	})
	assert.Contains(t, out, "Created sample config")

	content, err := os.ReadFile(filepath.Join(env.Dir, ".dirvana.yml"))
	require.NoError(t, err)
	for _, marker := range sampleConfigMarkers {
		assert.Contains(t, string(content), marker)
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  existing: echo existing\n")

	err := Init(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// The existing config is left untouched
	content, err := os.ReadFile(filepath.Join(env.Dir, ".dirvana.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "existing")
}

func TestInit_Global(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	out := captureStdout(t, func() {
		require.NoError(t, Init(true))
	})
	assert.Contains(t, out, "Created global config")

	content, err := os.ReadFile(filepath.Join(tmpDir, "dirvana", "global.yml"))
	require.NoError(t, err)
	for _, marker := range sampleConfigMarkers {
		assert.Contains(t, string(content), marker)
	}
}

func TestInit_Global_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	globalConfig := filepath.Join(tmpDir, "dirvana", "global.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(globalConfig), 0o755))
	require.NoError(t, os.WriteFile(globalConfig, []byte("aliases: {}\n"), 0o644))

	err := Init(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInit_GlobalWithoutHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	err := Init(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get global config path")
}

func TestInit_GlobalUncreatableDirectory(t *testing.T) {
	env := newTestEnv(t)

	// A regular file where the config directory would go
	blocker := filepath.Join(env.Root, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	t.Setenv("XDG_CONFIG_HOME", blocker)

	err := Init(true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config directory")
}

func TestInit_UnwritableDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	env := newTestEnv(t)
	require.NoError(t, os.Chmod(env.Dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(env.Dir, 0o755) })

	err := Init(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config file")
}

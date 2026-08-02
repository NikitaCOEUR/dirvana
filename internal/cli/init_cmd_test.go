package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

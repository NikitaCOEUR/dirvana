package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupLegacyHook(t *testing.T) {
	t.Run("removes block and preserves user content", func(t *testing.T) {
		tmpDir := t.TempDir()
		rcFile := filepath.Join(tmpDir, ".bashrc")
		content := "# Before\nexport FOO=bar\n\n" +
			legacyMarkerStart + "\n__dirvana_hook() { :; }\n" + legacyMarkerEnd + "\n\n# After\nalias g=git\n"
		require.NoError(t, os.WriteFile(rcFile, []byte(content), 0o644))

		removed, err := cleanupLegacyHook(rcFile)
		require.NoError(t, err)
		assert.True(t, removed)

		data, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		result := string(data)
		assert.Contains(t, result, "export FOO=bar")
		assert.Contains(t, result, "alias g=git")
		assert.NotContains(t, result, legacyMarkerStart)
		assert.NotContains(t, result, legacyMarkerEnd)
		assert.NotContains(t, result, "__dirvana_hook")
	})

	t.Run("no markers leaves file untouched", func(t *testing.T) {
		tmpDir := t.TempDir()
		rcFile := filepath.Join(tmpDir, ".bashrc")
		content := "# Just a regular bashrc\n"
		require.NoError(t, os.WriteFile(rcFile, []byte(content), 0o644))

		removed, err := cleanupLegacyHook(rcFile)
		require.NoError(t, err)
		assert.False(t, removed)

		data, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("missing file is not an error", func(t *testing.T) {
		removed, err := cleanupLegacyHook(filepath.Join(t.TempDir(), "nope"))
		require.NoError(t, err)
		assert.False(t, removed)
	})

	t.Run("block at the end of the file leaves the content above", func(t *testing.T) {
		tmpDir := t.TempDir()
		rcFile := filepath.Join(tmpDir, ".bashrc")
		content := "export FOO=bar\n\n" + legacyMarkerStart + "\n__dirvana_hook() { :; }\n" + legacyMarkerEnd + "\n"
		require.NoError(t, os.WriteFile(rcFile, []byte(content), 0o644))

		removed, err := cleanupLegacyHook(rcFile)
		require.NoError(t, err)
		assert.True(t, removed)

		data, err := os.ReadFile(rcFile)
		require.NoError(t, err)
		assert.Equal(t, "export FOO=bar\n", string(data))
	})

	t.Run("unreadable file is reported", func(t *testing.T) {
		// A directory where the RC file is expected
		rcFile := filepath.Join(t.TempDir(), "rc-dir")
		require.NoError(t, os.Mkdir(rcFile, 0o755))

		_, err := cleanupLegacyHook(rcFile)
		require.Error(t, err)
	})
}

func TestInstallHook_CleansLegacyInlineHook(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// RC file as left by a pre-strategy release: user content + inline hook
	rcFile := filepath.Join(tmpDir, ".bashrc")
	content := "# My bashrc\nexport PATH=$PATH:/usr/local/bin\n\n" +
		legacyMarkerStart + "\n__dirvana_hook() { :; }\nPROMPT_COMMAND=__dirvana_hook\n" + legacyMarkerEnd + "\n"
	require.NoError(t, os.WriteFile(rcFile, []byte(content), 0o644))

	result, err := InstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Contains(t, result.Message, "legacy inline hook")

	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	rc := string(data)

	// Old block gone, user content preserved, new hook referenced
	assert.NotContains(t, rc, legacyMarkerStart)
	assert.NotContains(t, rc, "PROMPT_COMMAND=__dirvana_hook")
	assert.Contains(t, rc, "# My bashrc")
	hookPath := filepath.Join(tmpDir, ".config", "dirvana", "hook-bash.sh")
	assert.Contains(t, rc, hookPath)
	_, err = os.Stat(hookPath)
	assert.NoError(t, err, "new external hook file should exist")
}

func TestUninstallHook_CleansLegacyInlineHook(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Only the legacy inline hook is present (new hook never installed)
	rcFile := filepath.Join(tmpDir, ".bashrc")
	content := "# Content before\n" +
		legacyMarkerStart + "\n# old hook\n" + legacyMarkerEnd + "\n# Content after\n"
	require.NoError(t, os.WriteFile(rcFile, []byte(content), 0o644))

	result, err := UninstallHook("bash")
	require.NoError(t, err)
	assert.True(t, result.Updated)
	assert.Contains(t, result.Message, "legacy inline hook")

	data, err := os.ReadFile(rcFile)
	require.NoError(t, err)
	rc := string(data)
	assert.NotContains(t, rc, legacyMarkerStart)
	assert.Contains(t, rc, "# Content before")
	assert.Contains(t, rc, "# Content after")
}

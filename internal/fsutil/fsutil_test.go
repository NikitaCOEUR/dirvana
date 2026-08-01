package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write new file
	content := []byte("test content")
	require.NoError(t, AtomicWrite(testFile, content, StateFilePerm))

	readContent, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Overwrite existing file
	newContent := []byte("new test content")
	require.NoError(t, AtomicWrite(testFile, newContent, StateFilePerm))

	readContent, err = os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, newContent, readContent)

	// No temp files left behind
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".dirvana-tmp-", "temporary file left behind")
	}
}

func TestAtomicWrite_InvalidDirectory(t *testing.T) {
	err := AtomicWrite("/nonexistent/path/that/does/not/exist/file.txt", []byte("test"), StateFilePerm)
	assert.Error(t, err)
}

func TestAtomicWrite_Permissions(t *testing.T) {
	tmpDir := t.TempDir()

	for name, perm := range map[string]os.FileMode{
		"state file 0600": StateFilePerm,
		"rc file 0644":    0644,
	} {
		t.Run(name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, strings.ReplaceAll(name, " ", "_"))
			require.NoError(t, AtomicWrite(testFile, []byte("data"), perm))

			info, err := os.Stat(testFile)
			require.NoError(t, err)
			assert.Equal(t, perm, info.Mode().Perm())
		})
	}
}

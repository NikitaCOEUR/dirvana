package fsutil

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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

func TestAtomicWrite_RenameOverDirectoryFails(t *testing.T) {
	// The destination is a directory: the final rename must fail and the
	// temp file must not be left behind
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	require.NoError(t, os.Mkdir(target, 0o755))

	err := AtomicWrite(target, []byte("data"), StateFilePerm)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rename")

	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".dirvana-tmp-", "temp file leaked after failed rename")
	}
}

// The failing chmod and close branches of AtomicWrite are deliberately
// left uncovered: fchmod and close on a local regular file the process
// just created do not fail short of a hardware or NFS error, and faking
// one would mean injecting a seam into production code that exists for
// no other reason. The write failure below is the one that does happen -
// a full filesystem - and it is covered.

func TestAtomicWrite_WriteFailure(t *testing.T) {
	// A full filesystem is the realistic way the write fails; a zero file
	// size limit reproduces it without needing a dedicated mount.
	// SIGXFSZ has to be ignored, otherwise exceeding the limit kills the
	// test binary instead of returning EFBIG.
	signal.Ignore(syscall.SIGXFSZ)
	t.Cleanup(func() { signal.Reset(syscall.SIGXFSZ) })

	var orig syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &orig); err != nil {
		t.Skipf("RLIMIT_FSIZE unavailable: %v", err)
	}
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 0, Max: orig.Max}); err != nil {
		t.Skipf("cannot lower RLIMIT_FSIZE: %v", err)
	}
	t.Cleanup(func() { _ = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &orig) })

	tmpDir := t.TempDir()
	err := AtomicWrite(filepath.Join(tmpDir, "state.json"), []byte("data"), StateFilePerm)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write to temp file")

	// The destination must not exist, and no temp file may be left over
	assert.NoFileExists(t, filepath.Join(tmpDir, "state.json"))
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".dirvana-tmp-", "temp file leaked after a failed write")
	}
}

func TestAtomicWrite_Permissions(t *testing.T) {
	tmpDir := t.TempDir()

	for name, perm := range map[string]os.FileMode{
		"state file 0600": StateFilePerm,
		"rc file 0644":    0o644,
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

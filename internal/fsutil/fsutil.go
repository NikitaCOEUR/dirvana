// Package fsutil provides filesystem helpers shared by packages that
// persist dirvana state (cache, auth, completion caches, shell hooks).
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// Permissions for dirvana state: files may reference private paths and
// feed shell evaluation, so they are restricted to the owning user.
const (
	// StateFilePerm is the mode for state files (cache, auth, downloaded scripts)
	StateFilePerm os.FileMode = 0o600
	// StateDirPerm is the mode for directories holding state files
	StateDirPerm os.FileMode = 0o700
)

// AtomicWrite writes data to filename atomically: the content is written
// to a temporary file in the same directory, then renamed over the
// destination. A crash mid-write can never leave a truncated file.
func AtomicWrite(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, ".dirvana-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	// Clean up temp file if something goes wrong
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Success - don't clean up temp file
	tmpFile = nil
	return nil
}

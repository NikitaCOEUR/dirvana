package fsutil

import (
	"path/filepath"
	"strings"
)

// NormalizePath cleans a path and drops any trailing separator, so that two
// spellings of the same name compare equal. It never touches the filesystem.
func NormalizePath(path string) string {
	cleaned := filepath.Clean(path)
	if cleaned == string(filepath.Separator) {
		// The root is all separator; trimming it would leave nothing
		return cleaned
	}
	return strings.TrimSuffix(cleaned, string(filepath.Separator))
}

// ResolvePath returns a path with every symlink along it resolved, which is
// what identifies a directory rather than one of the names leading to it.
//
// A shell keeps the logical path in $PWD and os.Getwd honours it, so the same
// directory reaches dirvana under whichever name the user typed - the one they
// authorised, or another one entirely.
//
// Falls back to NormalizePath when the path cannot be resolved: it may not
// exist yet, or sit behind a directory the user cannot traverse, and neither
// is a reason to refuse an answer.
func ResolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return NormalizePath(path)
	}
	return NormalizePath(resolved)
}

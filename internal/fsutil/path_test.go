package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/a/b", "/a/b"},
		{"/a/b/", "/a/b"},
		{"/a//b", "/a/b"},
		{"/a/./b", "/a/b"},
		{"/a/c/../b", "/a/b"},
		{"/", string(filepath.Separator)},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, NormalizePath(tt.input), "NormalizePath(%q)", tt.input)
	}
}

func TestResolvePath_FollowsSymlinks(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	link := filepath.Join(tmp, "link")
	require.NoError(t, os.Mkdir(target, 0o755))
	require.NoError(t, os.Symlink(target, link))

	// Both names have to land on the same answer: that is what makes an
	// authorization identify a directory rather than a way of reaching it
	assert.Equal(t, ResolvePath(target), ResolvePath(link))
	assert.Equal(t, ResolvePath(filepath.Join(link, "..", "target")), ResolvePath(target))
}

func TestResolvePath_FallsBackWhenItCannotResolve(t *testing.T) {
	// A path that does not exist yet still deserves an answer, cleaned
	assert.Equal(t, "/nowhere/at/all", ResolvePath("/nowhere/at//all/"))
}

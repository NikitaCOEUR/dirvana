package auth

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testProjectPath = "/test/project"

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)
	assert.NotNil(t, a)
}

func TestAuth_Allow(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	err = a.Allow(testProjectPath)
	require.NoError(t, err)

	// Verify it was authorized
	allowed, err := a.IsAllowed(testProjectPath)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAuth_IsAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	// Not authorized initially
	allowed, err := a.IsAllowed("/test/project")
	require.NoError(t, err)
	assert.False(t, allowed)

	// Authorize
	require.NoError(t, a.Allow("/test/project"))

	// Should be allowed now
	allowed, err = a.IsAllowed("/test/project")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAuth_Revoke(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	require.NoError(t, a.Allow(testProjectPath))

	// Verify it's allowed
	allowed, err := a.IsAllowed(testProjectPath)
	require.NoError(t, err)
	assert.True(t, allowed)

	// Revoke
	err = a.Revoke(testProjectPath)
	require.NoError(t, err)

	// Should not be allowed anymore
	allowed, err = a.IsAllowed(testProjectPath)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestAuth_List(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	// Initially empty
	list := a.List()
	assert.Empty(t, list)

	// Add some paths
	require.NoError(t, a.Allow("/test/project1"))
	require.NoError(t, a.Allow("/test/project2"))
	require.NoError(t, a.Allow("/test/project3"))

	list = a.List()
	assert.Len(t, list, 3)
	assert.Contains(t, list, "/test/project1")
	assert.Contains(t, list, "/test/project2")
	assert.Contains(t, list, "/test/project3")
}

func TestAuth_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	// Create auth and allow a path
	a1, err := New(authPath)
	require.NoError(t, err)
	require.NoError(t, a1.Allow("/test/project"))

	// Create new auth instance from same file
	a2, err := New(authPath)
	require.NoError(t, err)

	allowed, err := a2.IsAllowed("/test/project")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAuth_NormalizesPaths(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	// Allow with trailing slash
	require.NoError(t, a.Allow("/test/project/"))

	// Check without trailing slash
	allowed, err := a.IsAllowed("/test/project")
	require.NoError(t, err)
	assert.True(t, allowed)

	// Check with trailing slash
	allowed, err = a.IsAllowed("/test/project/")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAuth_InvalidPath(t *testing.T) {
	invalidPath := filepath.Join("/nonexistent", "path", "auth.json")
	_, err := New(invalidPath)
	assert.Error(t, err)
}

func TestAuth_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	// Add multiple paths
	require.NoError(t, a.Allow("/test/project1"))
	require.NoError(t, a.Allow("/test/project2"))

	// Clear all
	err = a.Clear()
	require.NoError(t, err)

	list := a.List()
	assert.Empty(t, list)
}

func TestAuth_AllowDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	testPath := "/test/project"

	// Allow same path multiple times
	require.NoError(t, a.Allow(testPath))
	require.NoError(t, a.Allow(testPath))
	require.NoError(t, a.Allow(testPath))

	// Should only appear once in list
	list := a.List()
	count := 0
	for _, p := range list {
		if p == testPath {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellApprovalFlow(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")
	a, err := New(authPath)
	require.NoError(t, err)

	dir := testProjectPath
	shellCmds := map[string]string{
		"GIT_BRANCH": "git rev-parse --abbrev-ref HEAD",
		"USER":       "whoami",
	}

	// Not allowed yet, should not require shell approval
	require.False(t, a.RequiresShellApproval(dir, shellCmds))

	// Allow directory
	require.NoError(t, a.Allow(dir))

	// Should require approval (never approved)
	require.True(t, a.RequiresShellApproval(dir, shellCmds))

	// Approve shell commands
	require.NoError(t, a.ApproveShellCommands(dir, shellCmds))

	// Should not require approval (already approved)
	require.False(t, a.RequiresShellApproval(dir, shellCmds))

	// Change shell commands (add new)
	shellCmds["BUILD_TIME"] = "date +%s"
	require.True(t, a.RequiresShellApproval(dir, shellCmds))

	// Approve new set
	require.NoError(t, a.ApproveShellCommands(dir, shellCmds))
	require.False(t, a.RequiresShellApproval(dir, shellCmds))

	// Remove a command (hash changes)
	delete(shellCmds, "USER")
	require.True(t, a.RequiresShellApproval(dir, shellCmds))
}

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

	testPath := testProjectPath

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

func TestAuth_AllowIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	a, err := New(authPath)
	require.NoError(t, err)

	testPath := testProjectPath

	// First allow - should persist
	require.NoError(t, a.Allow(testPath))

	// Get the original AllowedAt timestamp
	auth := a.GetAuth(testPath)
	require.NotNil(t, auth)
	originalTimestamp := auth.AllowedAt

	// Get the file modification time after first allow
	v2Path := filepath.Join(tmpDir, "authorized_v2.json")
	stat1, err := os.Stat(v2Path)
	require.NoError(t, err)
	modTime1 := stat1.ModTime()

	// Second allow - should be idempotent (no persist)
	require.NoError(t, a.Allow(testPath))

	// Timestamp should be preserved
	auth = a.GetAuth(testPath)
	require.NotNil(t, auth)
	assert.Equal(t, originalTimestamp, auth.AllowedAt, "AllowedAt should be preserved on idempotent call")

	// File should not have been modified
	stat2, err := os.Stat(v2Path)
	require.NoError(t, err)
	assert.Equal(t, modTime1, stat2.ModTime(), "File should not be modified on idempotent call")
}

func TestAuth_AllowAfterDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	// Create a V2 file with a directory that has Allowed=false
	v2Path := filepath.Join(tmpDir, "authorized_v2.json")
	v2Data := `{"_version":2,"directories":{"/test/project":{"allowed":false,"allowed_at":"2020-01-01T00:00:00Z"}}}`
	require.NoError(t, os.WriteFile(v2Path, []byte(v2Data), 0600))

	a, err := New(authPath)
	require.NoError(t, err)

	// Directory exists but is not allowed
	allowed, err := a.IsAllowed(testProjectPath)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Allow should update the existing entry
	require.NoError(t, a.Allow(testProjectPath))

	// Should now be allowed
	allowed, err = a.IsAllowed(testProjectPath)
	require.NoError(t, err)
	assert.True(t, allowed)

	// AllowedAt should be updated
	auth := a.GetAuth(testProjectPath)
	require.NotNil(t, auth)
	assert.True(t, auth.AllowedAt.After(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
}

func TestAuth_MigrationWithCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	// Write corrupted JSON in V1 file
	corruptedJSON := `{invalid json here`
	require.NoError(t, os.WriteFile(authPath, []byte(corruptedJSON), 0644))

	// Should succeed (start with empty state) but not load anything
	a, err := New(authPath)
	require.NoError(t, err)
	assert.Empty(t, a.List(), "Corrupted file should result in empty auth list")

	// Now write a corrupted V2 file
	v2Path := filepath.Join(tmpDir, "authorized_v2.json")
	require.NoError(t, os.WriteFile(v2Path, []byte(corruptedJSON), 0644))

	// V2 corruption should also be handled gracefully
	a2, err := New(authPath)
	require.NoError(t, err)
	assert.Empty(t, a2.List(), "Corrupted V2 file should result in empty auth list")
}

func TestAuth_MigrationFromV1Format(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	// Write V1 format ([]string) to file
	v1FormatJSON := `[
  "/home/user/project1",
  "/home/user/project2",
  "/home/user/project3"
]`
	require.NoError(t, os.WriteFile(authPath, []byte(v1FormatJSON), 0644))

	// Load with new Auth structure - should migrate automatically
	a, err := New(authPath)
	require.NoError(t, err)

	// Verify migration: all paths should be in the list
	list := a.List()
	assert.Len(t, list, 3)
	assert.Contains(t, list, "/home/user/project1")
	assert.Contains(t, list, "/home/user/project2")
	assert.Contains(t, list, "/home/user/project3")

	// Verify the migrated entries have proper DirAuth structure
	auth1 := a.GetAuth("/home/user/project1")
	require.NotNil(t, auth1)
	assert.True(t, auth1.Allowed)
	assert.False(t, auth1.AllowedAt.IsZero())

	// V1 file should still exist and be UNTOUCHED
	v1Data, err := os.ReadFile(authPath)
	require.NoError(t, err)
	assert.Equal(t, v1FormatJSON, string(v1Data), "V1 file should remain unchanged")

	// But when we persist changes, V2 file should be created
	require.NoError(t, a.Allow("/home/user/project4"))

	// V2 file should now exist with versioned format
	v2Path := filepath.Join(tmpDir, "authorized_v2.json")
	v2Data, err := os.ReadFile(v2Path)
	require.NoError(t, err)
	assert.Contains(t, string(v2Data), `"_version"`)
	assert.Contains(t, string(v2Data), `"directories"`)
	assert.Contains(t, string(v2Data), `"/home/user/project4"`)

	// V1 file should STILL be untouched
	v1DataAfter, err := os.ReadFile(authPath)
	require.NoError(t, err)
	assert.Equal(t, v1FormatJSON, string(v1DataAfter), "V1 file should never be modified")
}

func TestAuth_RequiresShellApproval_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")
	a, err := New(authPath)
	require.NoError(t, err)

	dir := testProjectPath

	t.Run("EmptyShellCommands", func(t *testing.T) {
		// Empty shell commands should not require approval
		require.False(t, a.RequiresShellApproval(dir, map[string]string{}))
		require.False(t, a.RequiresShellApproval(dir, nil))
	})

	t.Run("DirectoryNotInAuth", func(t *testing.T) {
		// Directory not in auth should not require approval (directory auth first)
		shellCmds := map[string]string{"USER": "whoami"}
		require.False(t, a.RequiresShellApproval(dir, shellCmds))
	})

	t.Run("DirectoryNotAllowed", func(t *testing.T) {
		// Directory exists but not allowed
		require.NoError(t, a.Revoke(dir)) // Ensure it's not allowed
		shellCmds := map[string]string{"USER": "whoami"}
		require.False(t, a.RequiresShellApproval(dir, shellCmds))
	})
}

func TestAuth_ApproveShellCommands_DirectoryNotAuthorized(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")
	a, err := New(authPath)
	require.NoError(t, err)

	dir := "/test/unauth-project"
	shellCmds := map[string]string{
		"USER": "whoami",
	}

	// Try to approve shell commands for a non-authorized directory
	err = a.ApproveShellCommands(dir, shellCmds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory not authorized")
}

func TestAuth_LoadV1_EdgeCases(t *testing.T) {
	t.Run("EmptyV1Array", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "authorized.json")

		// Write empty V1 array
		v1Data := []byte(`[]`)
		require.NoError(t, os.WriteFile(authPath, v1Data, 0600))

		a, err := New(authPath)
		require.NoError(t, err)
		assert.NotNil(t, a)
		assert.Empty(t, a.List())
	})

	t.Run("InvalidV1Format", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "authorized.json")

		// Write invalid JSON - New() handles this gracefully
		v1Data := []byte(`{invalid json}`)
		require.NoError(t, os.WriteFile(authPath, v1Data, 0600))

		// New() should succeed but start with empty state
		a, err := New(authPath)
		require.NoError(t, err)
		assert.NotNil(t, a)
		assert.Empty(t, a.List())
	})

	t.Run("EmptyFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "authorized.json")

		// Write empty file
		require.NoError(t, os.WriteFile(authPath, []byte(""), 0600))

		a, err := New(authPath)
		require.NoError(t, err)
		assert.NotNil(t, a)
		assert.Empty(t, a.List())
	})
}

func TestAuth_LoadV2_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized_v2.json")

	// Write V2 file with invalid version - New() handles this gracefully
	v2Data := `{"_version":99,"directories":{}}`
	require.NoError(t, os.WriteFile(authPath, []byte(v2Data), 0600))

	// New() should succeed but start with empty state
	a, err := New(filepath.Join(tmpDir, "authorized.json"))
	require.NoError(t, err)
	assert.NotNil(t, a)
	assert.Empty(t, a.List())
}

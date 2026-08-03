package auth

import (
	"fmt"
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
	stat1, err := os.Stat(authPath)
	require.NoError(t, err)
	modTime1 := stat1.ModTime()

	// Second allow - should be idempotent (no persist)
	require.NoError(t, a.Allow(testPath))

	// Timestamp should be preserved
	auth = a.GetAuth(testPath)
	require.NotNil(t, auth)
	assert.Equal(t, originalTimestamp, auth.AllowedAt, "AllowedAt should be preserved on idempotent call")

	// File should not have been modified
	stat2, err := os.Stat(authPath)
	require.NoError(t, err)
	assert.Equal(t, modTime1, stat2.ModTime(), "File should not be modified on idempotent call")
}

func TestAuth_AllowAfterDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized_v2.json")

	// Create an auth file with a directory that has Allowed=false
	authData := `{"_version":2,"directories":{"/test/project":{"allowed":false,"allowed_at":"2020-01-01T00:00:00Z"}}}`
	require.NoError(t, os.WriteFile(authPath, []byte(authData), 0o600))

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

func TestAuth_Load_EdgeCases(t *testing.T) {
	t.Run("LegacyV1ArrayIgnored", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "authorized_v2.json")

		// Legacy V1 format ([]string) is no longer supported and is
		// treated like any unreadable file: empty state.
		v1Data := []byte(`["/home/user/project1"]`)
		require.NoError(t, os.WriteFile(authPath, v1Data, 0o600))

		a, err := New(authPath)
		require.NoError(t, err)
		assert.NotNil(t, a)
		assert.Empty(t, a.List())
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "authorized_v2.json")

		require.NoError(t, os.WriteFile(authPath, []byte(`{invalid json}`), 0o600))

		// New() should succeed but start with empty state
		a, err := New(authPath)
		require.NoError(t, err)
		assert.NotNil(t, a)
		assert.Empty(t, a.List())
	})

	t.Run("EmptyFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "authorized_v2.json")

		require.NoError(t, os.WriteFile(authPath, []byte(""), 0o600))

		a, err := New(authPath)
		require.NoError(t, err)
		assert.NotNil(t, a)
		assert.Empty(t, a.List())
	})

	t.Run("UnsupportedVersion", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "authorized_v2.json")

		require.NoError(t, os.WriteFile(authPath, []byte(`{"_version":99,"directories":{}}`), 0o600))

		// New() should succeed but start with empty state
		a, err := New(authPath)
		require.NoError(t, err)
		assert.NotNil(t, a)
		assert.Empty(t, a.List())
	})
}

// symlinkedProject returns a directory and a symlink pointing at it, the shape
// of a project reached through ~/work -> /mnt/data/work.
func symlinkedProject(t *testing.T) (target, link string) {
	t.Helper()

	tmp := t.TempDir()
	target = filepath.Join(tmp, "project")
	link = filepath.Join(tmp, "link")
	require.NoError(t, os.Mkdir(target, 0o755))
	require.NoError(t, os.Symlink(target, link))
	return target, link
}

// TestAuth_SymlinkedPathIsTheSameProject covers what an authorization means: a
// directory, not the name used to reach it. Authorizing one spelling and
// arriving through the other used to be refused, with no way to tell why.
func TestAuth_SymlinkedPathIsTheSameProject(t *testing.T) {
	for _, tt := range []struct {
		name           string
		allowed, tried func(target, link string) string
	}{
		{
			"allow target, arrive by link",
			func(target, _ string) string { return target },
			func(_, link string) string { return link },
		},
		{
			"allow link, arrive by target",
			func(_, link string) string { return link },
			func(target, _ string) string { return target },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			target, link := symlinkedProject(t)
			a, err := New(filepath.Join(t.TempDir(), "authorized.json"))
			require.NoError(t, err)

			require.NoError(t, a.Allow(tt.allowed(target, link)))

			allowed, err := a.IsAllowed(tt.tried(target, link))
			require.NoError(t, err)
			assert.True(t, allowed)
		})
	}
}

func TestAuth_RevokeReachesBothSpellings(t *testing.T) {
	target, link := symlinkedProject(t)
	a, err := New(filepath.Join(t.TempDir(), "authorized.json"))
	require.NoError(t, err)

	require.NoError(t, a.Allow(link))
	require.NoError(t, a.Revoke(target))

	// Revoking has to mean revoked, whichever name either side used
	for _, path := range []string{target, link} {
		allowed, err := a.IsAllowed(path)
		require.NoError(t, err)
		assert.False(t, allowed, "still authorized through %s", path)
	}
}

// TestAuth_DoesNotFollowARepointedSymlink is the security half of resolving:
// an authorization is pinned to the directory it was granted for, so pointing
// the symlink somewhere else does not carry it over.
func TestAuth_DoesNotFollowARepointedSymlink(t *testing.T) {
	target, link := symlinkedProject(t)
	other := filepath.Join(filepath.Dir(target), "other")
	require.NoError(t, os.Mkdir(other, 0o755))

	a, err := New(filepath.Join(t.TempDir(), "authorized.json"))
	require.NoError(t, err)
	require.NoError(t, a.Allow(link))

	require.NoError(t, os.Remove(link))
	require.NoError(t, os.Symlink(other, link))

	allowed, err := a.IsAllowed(link)
	require.NoError(t, err)
	assert.False(t, allowed, "the authorization was granted for another directory")
}

// TestAuth_HonoursEntriesWrittenBeforeResolving keeps upgrades quiet: releases
// up to v0.10.0 filed entries under the literal path, and those must keep
// working rather than silently asking for authorization again.
func TestAuth_HonoursEntriesWrittenBeforeResolving(t *testing.T) {
	_, link := symlinkedProject(t)
	authPath := filepath.Join(t.TempDir(), "authorized.json")

	legacy := fmt.Sprintf(`{"_version": 2, "directories": {%q: {"allowed": true}}}`, link)
	require.NoError(t, os.WriteFile(authPath, []byte(legacy), 0o600))

	a, err := New(authPath)
	require.NoError(t, err)

	allowed, err := a.IsAllowed(link)
	require.NoError(t, err)
	assert.True(t, allowed)

	// And approving shell commands finds that entry rather than creating a
	// second one under the resolved name
	require.NoError(t, a.ApproveShellCommands(link, map[string]string{"BRANCH": "git branch"}))
	assert.Len(t, a.List(), 1)
}

func TestAuth_AllowIsIdempotentAcrossSpellings(t *testing.T) {
	target, link := symlinkedProject(t)
	a, err := New(filepath.Join(t.TempDir(), "authorized.json"))
	require.NoError(t, err)

	require.NoError(t, a.Allow(target))
	require.NoError(t, a.Allow(link))
	require.NoError(t, a.Allow(target+"/"))

	assert.Len(t, a.List(), 1, "one directory, one entry")
}

package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationIntegration tests the full migration flow from V1 to V2 (non-destructive)
func TestMigrationIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")

	// Step 1: Create a V1 format auth file (simulating dirvana v1)
	v1Format := []string{
		"/home/user/project-a",
		"/home/user/project-b",
		"/home/user/project-c",
	}
	v1Data, err := json.MarshalIndent(v1Format, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(authPath, v1Data, 0644))

	t.Log("Created V1 format auth file with 3 authorized projects")

	// Step 2: Initialize Auth (should load V1 read-only)
	auth, err := New(authPath)
	require.NoError(t, err, "Loading V1 should succeed without error")

	// Step 3: Verify the data was loaded correctly
	list := auth.List()
	assert.Len(t, list, 3, "Should have loaded 3 authorized projects")
	assert.Contains(t, list, "/home/user/project-a")
	assert.Contains(t, list, "/home/user/project-b")
	assert.Contains(t, list, "/home/user/project-c")

	// Step 4: Verify V1 file is still untouched
	v1DataAfter, err := os.ReadFile(authPath)
	require.NoError(t, err)
	assert.Equal(t, string(v1Data), string(v1DataAfter), "V1 file should remain unchanged")

	// Step 5: Make a change (this should create V2 file)
	require.NoError(t, auth.Allow("/home/user/project-d"))

	// Step 6: Verify V2 file was created
	v2Path := filepath.Join(tmpDir, "authorized_v2.json")
	v2Data, err := os.ReadFile(v2Path)
	require.NoError(t, err)

	var authFile File
	require.NoError(t, json.Unmarshal(v2Data, &authFile))
	assert.Equal(t, 2, authFile.Version, "Should have version 2")
	assert.Len(t, authFile.Directories, 4, "Should have all 4 projects now")
	assert.NotNil(t, authFile.Directories["/home/user/project-d"])

	// Step 7: Verify V1 is STILL untouched
	v1DataFinal, err := os.ReadFile(authPath)
	require.NoError(t, err)
	assert.Equal(t, string(v1Data), string(v1DataFinal), "V1 file should never be modified")

	// Step 8: Test shell approval on project
	shellCmds := map[string]string{"TEST": "echo test"}
	require.True(t, auth.RequiresShellApproval("/home/user/project-a", shellCmds))
	require.NoError(t, auth.ApproveShellCommands("/home/user/project-a", shellCmds))
	require.False(t, auth.RequiresShellApproval("/home/user/project-a", shellCmds))

	t.Log("✅ Migration completed successfully - V1 untouched, V2 created with version field")
} // TestNoMigrationNeeded tests that files already in V2 format are loaded correctly
func TestNoMigrationNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "authorized.json")
	v2Path := filepath.Join(tmpDir, "authorized_v2.json")

	// Create a file already in V2 format with version field
	authFile := File{
		Version: 2,
		Directories: map[string]*DirAuth{
			"/home/user/existing": {
				Allowed:           true,
				ShellCommandsHash: "abc123",
			},
		},
	}
	newData, err := json.MarshalIndent(authFile, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(v2Path, newData, 0644))

	// Load it (should use V2 directly, V1 doesn't exist)
	auth, err := New(authPath)
	require.NoError(t, err)

	// Verify data is intact
	dirAuth := auth.GetAuth("/home/user/existing")
	require.NotNil(t, dirAuth)
	assert.True(t, dirAuth.Allowed)
	assert.Equal(t, "abc123", dirAuth.ShellCommandsHash)
}

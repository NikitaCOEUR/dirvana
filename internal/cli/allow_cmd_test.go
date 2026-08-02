package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowWithParams_AlreadyAuthorizedIsIdempotent(t *testing.T) {
	env := newTestEnv(t)
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, AllowWithParams(AllowParams{
			AuthPath:    env.AuthPath,
			PathToAllow: env.Dir,
			LogLevel:    "error",
		}))
	})

	// Second authorization is a no-op, down to the absence of output
	assert.Empty(t, out)
}

func TestAllowWithParams_InvalidatesCache(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: ls -la\n")

	// A stale entry from before the authorization must not survive
	cacheStore, err := cache.New(env.CachePath)
	require.NoError(t, err)
	require.NoError(t, cacheStore.Set(&cache.Entry{
		Path:      env.Dir,
		Hash:      "stale",
		Timestamp: time.Now(),
		Version:   version.Version,
	}))

	out := captureStdout(t, func() {
		require.NoError(t, AllowWithParams(AllowParams{
			AuthPath:    env.AuthPath,
			PathToAllow: env.Dir,
			CachePath:   env.CachePath,
			LogLevel:    "error",
		}))
	})
	assert.Contains(t, out, "Authorized: "+env.Dir)
	// Current directory is the authorized one: the reload tip is shown
	assert.Contains(t, out, "dirvana export")

	reloaded, err := cache.New(env.CachePath)
	require.NoError(t, err)
	_, found := reloaded.Get(env.Dir)
	assert.False(t, found, "cache entry should have been invalidated")
}

func TestAllowWithParams_InvalidConfigFailsShellApproval(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: [unclosed\n")

	err := AllowWithParams(AllowParams{
		AuthPath:    env.AuthPath,
		PathToAllow: env.Dir,
		LogLevel:    "error",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")

	// The directory is authorized nonetheless: only the consent step failed
	authMgr, err := auth.New(env.AuthPath)
	require.NoError(t, err)
	allowed, err := authMgr.IsAllowed(env.Dir)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAllowWithParams_ReadOnlyAuthDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	env := newTestEnv(t)
	authDir := filepath.Join(env.Root, "state")
	require.NoError(t, os.Mkdir(authDir, 0o755))
	authPath := filepath.Join(authDir, "auth.json")

	// The auth file is created upfront, then writes are made impossible
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(env.Root))
	require.NoError(t, os.Chmod(authDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(authDir, 0o755) })

	err = AllowWithParams(AllowParams{
		AuthPath:    authPath,
		PathToAllow: env.Dir,
		LogLevel:    "error",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to authorize")
}

func TestHandleShellApproval_AlreadyApproved(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "env:\n  CURRENT_USER:\n    sh: whoami\n")

	// First pass approves the commands
	require.NoError(t, AllowWithParams(AllowParams{
		AuthPath:         env.AuthPath,
		PathToAllow:      env.Dir,
		AutoApproveShell: true,
		LogLevel:         "error",
	}))

	authMgr, err := auth.New(env.AuthPath)
	require.NoError(t, err)

	// Second pass: the hash is unchanged, no consent to ask for again
	out := captureStdout(t, func() {
		require.NoError(t, handleShellApproval(env.Dir, authMgr, false, testLogger()))
	})
	assert.Empty(t, out)
}

func TestHandleShellApproval_MergesInheritedCommands(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "env:\n  PARENT_VAR:\n    sh: echo parent\n")
	env.allow(t)

	child := filepath.Join(env.Dir, "child")
	env.writeConfigIn(t, child, "env:\n  CHILD_VAR:\n    sh: echo child\n")

	// Approving the child covers the commands inherited from the parent
	require.NoError(t, AllowWithParams(AllowParams{
		AuthPath:         env.AuthPath,
		PathToAllow:      child,
		AutoApproveShell: true,
		LogLevel:         "error",
	}))

	authMgr, err := auth.New(env.AuthPath)
	require.NoError(t, err)
	assert.False(t, authMgr.RequiresShellApproval(child, map[string]string{
		"PARENT_VAR": "echo parent",
		"CHILD_VAR":  "echo child",
	}))
}

func TestMergedShellEnv_NoConfig(t *testing.T) {
	env := newTestEnv(t)

	authMgr, err := auth.New(env.AuthPath)
	require.NoError(t, err)

	shellEnv, err := mergedShellEnv(env.Dir, authMgr)
	require.NoError(t, err)
	assert.Empty(t, shellEnv)
}

func TestRevokeWithParams_InvalidatesCacheOfSubdirectories(t *testing.T) {
	env := newTestEnv(t)
	child := filepath.Join(env.Dir, "child")
	require.NoError(t, os.MkdirAll(child, 0o755))
	env.allow(t)

	cacheStore, err := cache.New(env.CachePath)
	require.NoError(t, err)
	for _, dir := range []string{env.Dir, child} {
		require.NoError(t, cacheStore.Set(&cache.Entry{
			Path:      dir,
			Hash:      "hash",
			Timestamp: time.Now(),
			Version:   version.Version,
		}))
	}

	out := captureStdout(t, func() {
		require.NoError(t, RevokeWithParams(RevokeParams{
			AuthPath:     env.AuthPath,
			PathToRevoke: env.Dir,
			CachePath:    env.CachePath,
			LogLevel:     "error",
		}))
	})
	assert.Contains(t, out, "Revoked: "+env.Dir)
	// Current directory is the revoked one: the unload tip is shown
	assert.Contains(t, out, "cd -")

	reloaded, err := cache.New(env.CachePath)
	require.NoError(t, err)
	for _, dir := range []string{env.Dir, child} {
		_, found := reloaded.Get(dir)
		assert.False(t, found, "cache entry for %s should have been invalidated", dir)
	}
}

func TestRevokeWithParams_ReadOnlyAuthDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	env := newTestEnv(t)
	authDir := filepath.Join(env.Root, "state")
	require.NoError(t, os.Mkdir(authDir, 0o755))
	authPath := filepath.Join(authDir, "auth.json")

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(env.Dir))
	require.NoError(t, os.Chmod(authDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(authDir, 0o755) })

	err = RevokeWithParams(RevokeParams{
		AuthPath:     authPath,
		PathToRevoke: env.Dir,
		LogLevel:     "error",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to revoke")
}

func TestList_InvalidAuthPath(t *testing.T) {
	env := newTestEnv(t)

	// A regular file where the state directory is expected: it cannot be
	// created, whoever runs the test
	blocker := filepath.Join(env.Root, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := List(filepath.Join(blocker, "auth.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestList_Empty(t *testing.T) {
	env := newTestEnv(t)

	out := captureStdout(t, func() {
		require.NoError(t, List(env.AuthPath))
	})
	assert.Contains(t, out, "No authorized projects")
}

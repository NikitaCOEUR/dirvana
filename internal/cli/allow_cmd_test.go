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

// testPathConst is a path that never exists on disk: authorization is
// bookkeeping, it does not require the directory to be there
const testPathConst = "/test/path"

func TestAllow(t *testing.T) {
	env := newTestEnv(t)

	require.NoError(t, Allow(env.AuthPath, testPathConst))

	authMgr, err := auth.New(env.AuthPath)
	require.NoError(t, err)
	allowed, err := authMgr.IsAllowed(testPathConst)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAllow_EmptyPath(t *testing.T) {
	env := newTestEnv(t)

	// The auth layer stores whatever it is given, empty path included
	require.NoError(t, Allow(env.AuthPath, ""))
}

func TestAllow_InvalidAuthPath(t *testing.T) {
	env := newTestEnv(t)

	// A regular file where the state directory is expected
	blocker := filepath.Join(env.Root, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := Allow(filepath.Join(blocker, "auth.json"), testPathConst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestRevoke(t *testing.T) {
	env := newTestEnv(t)
	require.NoError(t, Allow(env.AuthPath, testPathConst))

	require.NoError(t, Revoke(env.AuthPath, testPathConst))

	authMgr, err := auth.New(env.AuthPath)
	require.NoError(t, err)
	allowed, err := authMgr.IsAllowed(testPathConst)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRevoke_NotAuthorized(t *testing.T) {
	env := newTestEnv(t)

	// Revoking what was never authorized is a no-op, not an error
	require.NoError(t, Revoke(env.AuthPath, testPathConst))
}

func TestRevoke_InvalidAuthPath(t *testing.T) {
	env := newTestEnv(t)

	blocker := filepath.Join(env.Root, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := Revoke(filepath.Join(blocker, "auth.json"), testPathConst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestList_ShowsAuthorizedPaths(t *testing.T) {
	env := newTestEnv(t)
	require.NoError(t, Allow(env.AuthPath, "/test/path1"))
	require.NoError(t, Allow(env.AuthPath, "/test/path2"))

	out := captureStdout(t, func() {
		require.NoError(t, List(env.AuthPath))
	})

	assert.Contains(t, out, "Authorized projects:")
	assert.Contains(t, out, "/test/path1")
	assert.Contains(t, out, "/test/path2")
}

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

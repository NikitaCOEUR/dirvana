package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
	"github.com/stretchr/testify/require"
)

// testEnv is an isolated workspace for CLI tests: its own cache and auth
// state files plus a work directory holding the config under test. The
// process is chdir'd into that directory, since the commands resolve their
// context from the current working directory.
type testEnv struct {
	Root      string
	Dir       string
	CachePath string
	AuthPath  string
}

// newTestEnv creates the workspace and enters its work directory for the
// duration of the test
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Resolve symlinks so the paths match what the commands see (on macOS
	// the temp dir lives behind /private)
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	env := &testEnv{
		Root:      root,
		Dir:       filepath.Join(root, "work"),
		CachePath: filepath.Join(root, "cache.json"),
		AuthPath:  filepath.Join(root, "auth.json"),
	}
	require.NoError(t, os.MkdirAll(env.Dir, 0o755))
	chdir(t, env.Dir)

	return env
}

// writeConfig writes a .dirvana.yml in the work directory
func (e *testEnv) writeConfig(t *testing.T, content string) {
	t.Helper()
	e.writeConfigIn(t, e.Dir, content)
}

// writeConfigIn writes a .dirvana.yml in an arbitrary directory of the
// hierarchy, creating it when needed
func (e *testEnv) writeConfigIn(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".dirvana.yml"), []byte(content), 0o644))
}

// allow authorizes the work directory so its config is actually loaded
func (e *testEnv) allow(t *testing.T) {
	t.Helper()
	e.allowDir(t, e.Dir)
}

// allowDir authorizes an arbitrary directory of the hierarchy
func (e *testEnv) allowDir(t *testing.T, dir string) {
	t.Helper()
	authMgr, err := auth.New(e.AuthPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(dir))
}

// mockTool writes an executable script in the workspace and returns its
// absolute path, usable as an alias command without touching PATH
func (e *testEnv) mockTool(t *testing.T, name, script string) string {
	t.Helper()
	binDir := filepath.Join(e.Root, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	path := filepath.Join(binDir, name)
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}

// testLogger returns a logger quiet enough not to pollute the test output
func testLogger() *logger.Logger {
	return logger.New("error", io.Discard)
}

// chdir enters dir and returns to the previous directory when the test ends
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// captureStdout runs fn with os.Stdout redirected and returns what it wrote
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = w

	// Drain concurrently so a write larger than the pipe buffer cannot
	// deadlock fn
	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		done <- string(out)
	}()

	defer func() {
		os.Stdout = orig
		_ = r.Close()
	}()

	fn()

	require.NoError(t, w.Close())
	return <-done
}

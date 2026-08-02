package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExec_NoCacheEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Create empty cache
	_, err := cache.New(cachePath)
	require.NoError(t, err)

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		LogLevel:  "error",
		Alias:     "test",
		Args:      []string{},
	}

	err = Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no dirvana context found")
}

func TestExec_AliasNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	// Create a real config file with an alias
	configPath := filepath.Join(workDir, ".dirvana.yml")
	configContent := `aliases:
  other: echo other
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Authorize the directory
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(workDir))

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		AuthPath:  authPath,
		LogLevel:  "error",
		Alias:     "nonexistent",
		Args:      []string{},
	}

	err = Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "alias 'nonexistent' not found")
}

func TestExec_EmptyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	// Create a real config file with empty command
	configPath := filepath.Join(workDir, ".dirvana.yml")
	configContent := `aliases:
  empty: ""
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

	// Authorize the directory
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(workDir))

	// Change to work directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	err = os.Chdir(workDir)
	require.NoError(t, err)

	params := ExecParams{
		CachePath: cachePath,
		AuthPath:  authPath,
		LogLevel:  "error",
		Alias:     "empty",
		Args:      []string{},
	}

	err = Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
}

// stubExecve replaces the process-replacing execve seam for the duration of
// a test and records the invocation. The real syscall.Exec would swallow
// the test binary (remaining tests and the coverage flush included).
func stubExecve(t *testing.T) *execveCall {
	t.Helper()
	call := &execveCall{}
	orig := execve
	execve = func(argv0 string, argv []string, envv []string) error {
		call.argv0 = argv0
		call.argv = argv
		call.envv = envv
		call.called = true
		return nil
	}
	t.Cleanup(func() { execve = orig })
	return call
}

type execveCall struct {
	called bool
	argv0  string
	argv   []string
	envv   []string
}

func TestExec_ExecutesResolvedCommandWithArgs(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	// Resolve symlinks for macOS compatibility
	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	configPath := filepath.Join(workDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("aliases:\n  ll: ls -la\n"), 0o644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(workDir))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(workDir))

	call := stubExecve(t)

	err = Exec(ExecParams{
		CachePath: cachePath,
		AuthPath:  authPath,
		LogLevel:  "error",
		Alias:     "ll",
		Args:      []string{"/tmp"},
	})
	// The stub returns nil, so Exec falls through to its unreachable-in-
	// production error path; what matters is what was about to be executed
	require.True(t, call.called, "exec must be invoked")
	assert.Error(t, err, "a returning execve means the exec failed")

	// The resolved command runs through the user's shell with args appended
	assert.NotEmpty(t, call.argv0)
	joined := strings.Join(call.argv, " ")
	assert.Contains(t, joined, "ls -la")
	assert.Contains(t, joined, "/tmp")
	assert.NotEmpty(t, call.envv)
}

// execParams builds the params for an alias of the workspace
func (e *testEnv) execParams(alias string, args ...string) ExecParams {
	return ExecParams{
		CachePath: e.CachePath,
		AuthPath:  e.AuthPath,
		LogLevel:  "error",
		Alias:     alias,
		Args:      args,
	}
}

func TestExec_RunsFunctionBody(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "functions:\n  greet: |\n    echo hello $1\n")
	env.allow(t)

	call := stubExecve(t)
	_ = Exec(env.execParams("greet", "world"))

	require.True(t, call.called)
	joined := strings.Join(call.argv, " ")
	// The function body is passed to the shell, marker prefix included
	assert.Contains(t, joined, "echo hello $1")
	assert.Contains(t, joined, "world")
}

func TestExec_ConditionMetUsesMainCommand(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `aliases:
  cond:
    command: echo "condition-met"
    when:
      file: marker.txt
    else: echo "condition-missing"
`)
	env.allow(t)
	require.NoError(t, os.WriteFile(filepath.Join(env.Dir, "marker.txt"), []byte("x"), 0o644))

	call := stubExecve(t)
	_ = Exec(env.execParams("cond"))

	require.True(t, call.called)
	assert.Contains(t, strings.Join(call.argv, " "), "condition-met")
}

func TestExec_ConditionNotMetWithoutFallback(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `aliases:
  cond:
    command: echo "condition-met"
    when:
      var: DIRVANA_TEST_UNSET_VAR
`)
	env.allow(t)

	// Without an else branch the main command stays selected
	call := stubExecve(t)
	_ = Exec(env.execParams("cond"))

	require.True(t, call.called)
	assert.Contains(t, strings.Join(call.argv, " "), "condition-met")
}

func TestExec_UnparsableConditionFallsBackToMainCommand(t *testing.T) {
	env := newTestEnv(t)
	// Mixing an atomic condition with a composite one is rejected by the
	// condition parser
	env.writeConfig(t, `aliases:
  cond:
    command: echo "main"
    when:
      file: marker.txt
      all:
        - var: SOME_VAR
`)
	env.allow(t)

	call := stubExecve(t)
	_ = Exec(env.execParams("cond"))

	require.True(t, call.called)
	assert.Contains(t, strings.Join(call.argv, " "), "main")
}

func TestExec_CompletionCallUsesCompletionCommand(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `aliases:
  k:
    command: kubectl --context prod
    completion: kubectl
`)
	env.allow(t)

	for _, arg := range []string{"__complete", "completion"} {
		t.Run(arg, func(t *testing.T) {
			call := stubExecve(t)
			_ = Exec(env.execParams("k", arg, "get"))

			require.True(t, call.called)
			joined := strings.Join(call.argv, " ")
			// The completion command replaces the alias command, so the
			// completion protocol is not confused by the extra flags
			assert.Contains(t, joined, "kubectl")
			assert.Contains(t, joined, arg)
			assert.NotContains(t, joined, "--context prod")
		})
	}
}

func TestExec_ShellNotFound(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: ls -la\n")
	env.allow(t)

	// No PATH, no shell to hand the command over to
	t.Setenv("PATH", "")

	err := Exec(env.execParams("ll"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shell not found")
}

func TestExec_ConditionalAliasUsesElseBranch(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")
	workDir := filepath.Join(tmpDir, "work")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	workDir, err := filepath.EvalSymlinks(workDir)
	require.NoError(t, err)

	configContent := `aliases:
  cond:
    command: echo "condition-met"
    when:
      file: marker.txt
    else: echo "condition-missing"
`
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ".dirvana.yml"), []byte(configContent), 0o644))

	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	require.NoError(t, authMgr.Allow(workDir))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(workDir))

	// No marker.txt: the else command must be selected
	call := stubExecve(t)
	_ = Exec(ExecParams{
		CachePath: cachePath,
		AuthPath:  authPath,
		LogLevel:  "error",
		Alias:     "cond",
	})
	require.True(t, call.called)
	assert.Contains(t, strings.Join(call.argv, " "), "condition-missing")
}

func TestExec_InvalidCachePath(t *testing.T) {
	params := ExecParams{
		CachePath: "/nonexistent/path/cache.json",
		LogLevel:  "error",
		Alias:     "test",
		Args:      []string{},
	}

	err := Exec(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

func TestExec_CacheLoadFailure(t *testing.T) {
	// Use a directory path as cache path (will fail to load)
	tmpDir := t.TempDir()

	params := ExecParams{
		CachePath: tmpDir, // Directory, not a file
		LogLevel:  "error",
		Alias:     "test",
		Args:      []string{},
	}

	err := Exec(params)
	assert.Error(t, err)
	// Should fail at cache loading
}

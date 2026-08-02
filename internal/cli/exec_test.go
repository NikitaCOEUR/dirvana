package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// command returns what the shell was asked to run
func (c *execveCall) command() string {
	return strings.Join(c.argv, " ")
}

func TestExec_NoContext(t *testing.T) {
	env := newTestEnv(t)

	err := Exec(env.execParams("test"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no dirvana context found")
}

func TestExec_AliasNotFound(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  other: echo other\n")
	env.allow(t)

	err := Exec(env.execParams("nonexistent"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alias 'nonexistent' not found")
}

func TestExec_EmptyCommand(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  empty: \"\"\n")
	env.allow(t)

	err := Exec(env.execParams("empty"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty command")
}

func TestExec_ExecutesResolvedCommandWithArgs(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: ls -la\n")
	env.allow(t)

	call := stubExecve(t)
	err := Exec(env.execParams("ll", "/tmp"))

	// The stub returns nil, so Exec falls through to its unreachable-in-
	// production error path; what matters is what was about to be executed
	require.True(t, call.called, "exec must be invoked")
	assert.Error(t, err, "a returning execve means the exec failed")

	// The resolved command runs through the user's shell with args appended
	assert.NotEmpty(t, call.argv0)
	assert.Contains(t, call.command(), "ls -la")
	assert.Contains(t, call.command(), "/tmp")
	assert.NotEmpty(t, call.envv)
}

func TestExec_RunsFunctionBody(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "functions:\n  greet: |\n    echo hello $1\n")
	env.allow(t)

	call := stubExecve(t)
	_ = Exec(env.execParams("greet", "world"))

	require.True(t, call.called)
	// The function body is passed to the shell, marker prefix included
	assert.Contains(t, call.command(), "echo hello $1")
	assert.Contains(t, call.command(), "world")
}

func TestExec_Conditions(t *testing.T) {
	const conditionalConfig = `aliases:
  cond:
    command: echo "condition-met"
    when:
      file: marker.txt
    else: echo "condition-missing"
`

	t.Run("met uses the main command", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, conditionalConfig)
		env.allow(t)
		require.NoError(t, os.WriteFile(filepath.Join(env.Dir, "marker.txt"), []byte("x"), 0o644))

		call := stubExecve(t)
		_ = Exec(env.execParams("cond"))

		require.True(t, call.called)
		assert.Contains(t, call.command(), "condition-met")
	})

	t.Run("unmet uses the else branch", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, conditionalConfig)
		env.allow(t)

		call := stubExecve(t)
		_ = Exec(env.execParams("cond"))

		require.True(t, call.called)
		assert.Contains(t, call.command(), "condition-missing")
	})

	t.Run("unmet without else keeps the main command", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, `aliases:
  cond:
    command: echo "condition-met"
    when:
      var: DIRVANA_TEST_UNSET_VAR
`)
		env.allow(t)

		call := stubExecve(t)
		_ = Exec(env.execParams("cond"))

		require.True(t, call.called)
		assert.Contains(t, call.command(), "condition-met")
	})

	t.Run("unparsable falls back to the main command", func(t *testing.T) {
		env := newTestEnv(t)
		// Mixing an atomic condition with a composite one is rejected by
		// the condition parser
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
		assert.Contains(t, call.command(), "main")
	})
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
			// The completion command replaces the alias command, so the
			// completion protocol is not confused by the extra flags
			assert.Contains(t, call.command(), "kubectl")
			assert.Contains(t, call.command(), arg)
			assert.NotContains(t, call.command(), "--context prod")
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

func TestExec_UnusableCachePath(t *testing.T) {
	env := newTestEnv(t)

	// A directory where the cache file is expected
	err := Exec(ExecParams{
		CachePath: env.Dir,
		AuthPath:  env.AuthPath,
		LogLevel:  "error",
		Alias:     "test",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

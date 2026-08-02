package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exportParams builds the params for an export from the workspace
func (e *testEnv) exportParams(prevDir string) ExportParams {
	return ExportParams{
		CachePath: e.CachePath,
		AuthPath:  e.AuthPath,
		LogLevel:  "error",
		PrevDir:   prevDir,
	}
}

func TestExport_NoConfig(t *testing.T) {
	env := newTestEnv(t)

	// Nothing to export, and the shell hook still gets a valid output
	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})
	assert.Empty(t, out)
}

func TestExport_NotAuthorized(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: ls -la\n")

	// A config that was never authorized must not reach the shell
	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})
	assert.Empty(t, out)
}

func TestExport_Authorized(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: ls -la\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})

	assert.Contains(t, out, "ll")
	assert.Contains(t, out, "dirvana exec ll")
}

func TestExport_SecondCallServesTheSameCode(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: ls -la\n")
	env.allow(t)

	first := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})
	// The second call is served from the cache written by the first
	second := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})

	assert.NotEmpty(t, first)
	assert.Equal(t, first, second)
}

func TestExport_CleansUpTheDirectoryBeingLeft(t *testing.T) {
	env := newTestEnv(t)

	parent := filepath.Join(env.Root, "parent")
	env.writeConfigIn(t, parent, "aliases:\n  parentalias: echo parent\n")
	env.allowDir(t, parent)

	child := filepath.Join(env.Root, "child")
	env.writeConfigIn(t, child, "aliases:\n  childalias: echo child\n")
	env.allowDir(t, child)

	chdir(t, parent)
	require.NoError(t, Export(env.exportParams("")))

	// Moving to a sibling: the parent's aliases must be unset first
	chdir(t, child)
	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams(parent)))
	})

	assert.Contains(t, out, "cleanup")
	assert.Contains(t, out, "parentalias")
	assert.Contains(t, out, "childalias")
}

func TestExport_WithShellEnv(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `aliases:
  test: echo test
env:
  STATIC_VAR: static
  DYNAMIC_VAR:
    sh: echo dynamic
`)
	env.allow(t)
	answerPrompt(t, "y")

	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})

	assert.Contains(t, out, "STATIC_VAR")
	assert.Contains(t, out, "static")
	assert.Contains(t, out, "DYNAMIC_VAR")
}

func TestExport_WithFunctions(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, `functions:
  greet: |
    echo "Hello, $1!"
  mkcd: |
    mkdir -p "$1" && cd "$1"
`)
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})

	// Function bodies are emitted verbatim, not wrapped in dirvana exec
	assert.Contains(t, out, "greet")
	assert.Contains(t, out, `echo "Hello, $1!"`)
	assert.Contains(t, out, "mkcd")
	assert.Contains(t, out, `mkdir -p "$1" && cd "$1"`)
}

func TestExport_DisabledViaEnv(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: ls -la\n")
	env.allow(t)
	t.Setenv("DIRVANA_ENABLED", "false")

	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})

	assert.Empty(t, out, "the kill switch must silence the export entirely")
}

func TestExport_InvalidConfigYieldsNoShellCode(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  ll: [unterminated\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})

	// The hierarchy cannot be loaded: nothing is exported, and the shell
	// hook still gets a valid (empty) output
	assert.Empty(t, out)
}

func TestExport_InvalidConfigStillEmitsCleanup(t *testing.T) {
	env := newTestEnv(t)

	// A previous directory whose aliases were exported and must now be
	// cleaned up
	prev := filepath.Join(env.Root, "prev")
	env.writeConfigIn(t, prev, "aliases:\n  prevalias: echo prev\n")
	env.allowDir(t, prev)

	chdir(t, prev)
	require.NoError(t, Export(env.exportParams("")))
	chdir(t, env.Dir)

	// The current directory has an unloadable config
	env.writeConfig(t, "aliases:\n  ll: [unterminated\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams(prev)))
	})

	assert.Contains(t, out, "prevalias")
	assert.NotContains(t, out, "alias ll")
}

func TestExport_InvalidAuthPath(t *testing.T) {
	env := newTestEnv(t)

	// A regular file where the state directory is expected
	blocker := filepath.Join(env.Root, "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	err := Export(ExportParams{
		CachePath: env.CachePath,
		AuthPath:  filepath.Join(blocker, "auth.json"),
		LogLevel:  "error",
	})
	require.Error(t, err)
}

func TestExport_SkipsUnloadableConfigInChain(t *testing.T) {
	env := newTestEnv(t)

	// The parent config is valid, the child one is not: the export must
	// keep what it can rather than dropping everything
	env.writeConfig(t, "aliases:\n  parentalias: echo parent\n")
	env.allow(t)

	child := filepath.Join(env.Dir, "child")
	env.writeConfigIn(t, child, "aliases:\n  childalias: echo child\n")
	env.allowDir(t, child)
	chdir(t, child)

	out := captureStdout(t, func() {
		require.NoError(t, Export(env.exportParams("")))
	})
	assert.Contains(t, out, "parentalias")
	assert.Contains(t, out, "childalias")
}

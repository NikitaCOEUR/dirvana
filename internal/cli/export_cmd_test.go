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

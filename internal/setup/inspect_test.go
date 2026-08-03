package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInspectHook_MovedBinaryIsNotOutdated is the point of matchHook: the
// dirvana path is baked into the hook at install time, so comparing the file
// byte for byte against freshly generated code calls every hook installed from
// another location outdated. Status showed that warning, and its one-key fix
// would then rewrite the hook to whatever path the running binary happened to
// have - a download directory, say, breaking every new shell once deleted.
func TestInspectHook_MovedBinaryIsNotOutdated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// A hook installed from a dirvana that lives somewhere else entirely
	installed, err := shell.GenerateHookCode("fish", "/usr/local/bin/dirvana")
	require.NoError(t, err)
	writeFishHook(t, home, installed)

	state, err := InspectHook("fish")
	require.NoError(t, err)

	assert.True(t, state.Installed)
	assert.False(t, state.Outdated, "a hook from another path is not out of date")
	assert.Equal(t, "/usr/local/bin/dirvana", state.BinaryPath)
}

func TestInspectHook_ChangedHookCodeIsOutdated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeFishHook(t, home, "function __dirvana_hook --on-variable PWD\n  # an older release\nend\n")

	state, err := InspectHook("fish")
	require.NoError(t, err)

	assert.True(t, state.Installed)
	assert.True(t, state.Outdated)
}

// TestInspectHook_MissingBinaryIsBroken covers the one case where a hook is
// silently dead: it still runs, and fails on every cd.
func TestInspectHook_MissingBinaryIsBroken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	installed, err := shell.GenerateHookCode("fish", filepath.Join(home, "gone", "dirvana"))
	require.NoError(t, err)
	writeFishHook(t, home, installed)

	state, err := InspectHook("fish")
	require.NoError(t, err)

	assert.True(t, state.Broken)
	assert.False(t, state.Outdated, "the code is current; only the binary is gone")
}

func TestInspectHook_PresentBinaryIsNotBroken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	binary := filepath.Join(home, "bin", "dirvana")
	require.NoError(t, os.MkdirAll(filepath.Dir(binary), 0o755))
	require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755))

	installed, err := shell.GenerateHookCode("fish", binary)
	require.NoError(t, err)
	writeFishHook(t, home, installed)

	state, err := InspectHook("fish")
	require.NoError(t, err)

	assert.False(t, state.Broken)
	assert.False(t, state.Outdated)
}

// TestInspectHook_FindsAnInstallFromTheOtherStrategy is why InspectHook asks
// every strategy: SelectInstallStrategy returns the one that would be used
// today, so a working hook installed the other way read as missing - status
// then claimed aliases were not loaded while they were.
func TestInspectHook_FindsAnInstallFromTheOtherStrategy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// An external-hook install, plus an RC file that mentions .bashrc.d, which
	// is all it takes for the drop-in strategy to be preferred
	external, err := NewExternalHookStrategy("bash")
	require.NoError(t, err)
	require.NoError(t, external.Install())

	rc := filepath.Join(home, ".bashrc")
	data, err := os.ReadFile(rc)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(rc, append([]byte("for f in ~/.bashrc.d/*; do source $f; done\n"), data...), 0o644))

	dropIn, err := NewDropInStrategy("bash")
	require.NoError(t, err)
	require.True(t, dropIn.IsSupported(), "the drop-in strategy must be the preferred one for this test to mean anything")
	require.False(t, dropIn.IsInstalled())

	state, err := InspectHook("bash")
	require.NoError(t, err)

	assert.True(t, state.Installed, "the external hook is installed and works")
	assert.Equal(t, external.HookFile(), state.File)
}

// TestInspectHook_ReportsTheFileHoldingTheHook keeps status from sending the
// user to a file with no dirvana code in it.
func TestInspectHook_ReportsTheFileHoldingTheHook(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	require.NoError(t, os.WriteFile(filepath.Join(home, ".bashrc"),
		[]byte("for f in ~/.bashrc.d/*; do source $f; done\n"), 0o644))

	dropIn, err := NewDropInStrategy("bash")
	require.NoError(t, err)
	require.NoError(t, dropIn.Install())

	state, err := InspectHook("bash")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(home, ".bashrc.d", "dirvana.sh"), state.File)
	assert.Equal(t, filepath.Join(home, ".bashrc"), state.RCFile)
}

func TestInspectHook_NothingInstalled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	state, err := InspectHook("zsh")
	require.NoError(t, err)

	assert.False(t, state.Installed)
	assert.False(t, state.Outdated)
	assert.False(t, state.Broken)
	// Where it would go, so status can still name it
	assert.NotEmpty(t, state.File)
	assert.NotEmpty(t, state.RCFile)
}

func TestInspectHook_UnsupportedShell(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := InspectHook("nushell")
	assert.Error(t, err)
}

func TestMatchHook_RejectsTruncatedHook(t *testing.T) {
	pattern, err := shell.GenerateHookCode("bash", "/usr/bin/dirvana")
	require.NoError(t, err)

	_, matches := matchHook("bash", pattern[:len(pattern)/2])
	assert.False(t, matches)
}

func TestBinaryExists(t *testing.T) {
	assert.True(t, binaryExists("sh"), "a bare name resolves through PATH")
	assert.False(t, binaryExists("definitely-not-a-command-xyz"))
	assert.False(t, binaryExists("/nonexistent/dirvana"))
	assert.False(t, binaryExists(t.TempDir()), "a directory is not a binary")
}

// writeFishHook installs hook content the way FishHookStrategy would, so
// IsInstalled recognises it.
func writeFishHook(t *testing.T, home, content string) {
	t.Helper()

	hookPath := filepath.Join(home, ".config", "dirvana", "hook-fish.sh")
	require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o755))
	require.NoError(t, os.WriteFile(hookPath, []byte(content), 0o644))

	rcFile := filepath.Join(home, ".config", "fish", "config.fish")
	require.NoError(t, os.MkdirAll(filepath.Dir(rcFile), 0o755))
	require.NoError(t, os.WriteFile(rcFile,
		[]byte("if status is-interactive\n    source "+hookPath+"\nend\n"), 0o644))
}

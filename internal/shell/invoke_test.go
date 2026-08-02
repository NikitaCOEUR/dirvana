package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutable(t *testing.T) {
	assert.Equal(t, "bash", Executable(Bash))
	assert.Equal(t, "zsh", Executable(Zsh))
	assert.Equal(t, "fish", Executable(Fish))
	// Unknown shells fall back to bash (sh is not supported)
	assert.Equal(t, "bash", Executable("ksh"))
}

func TestBuildArgs_FishWithFlags(t *testing.T) {
	// Fish must use -- to prevent interpreting user args as fish flags
	argv := BuildArgs("fish", Fish, "talosctl", []string{"-n", "10.0.0.1", "get", "disks"})

	// Should be: fish --no-config -c "talosctl $argv" -- -n 10.0.0.1 get disks
	assert.Equal(t, "fish", argv[0])
	assert.Equal(t, "--no-config", argv[1])
	assert.Equal(t, "-c", argv[2])
	assert.Equal(t, "talosctl $argv", argv[3])
	assert.Equal(t, "--", argv[4], "fish must have -- end-of-options marker")
	assert.Equal(t, "-n", argv[5])
	assert.Equal(t, "10.0.0.1", argv[6])
	assert.Equal(t, "get", argv[7])
	assert.Equal(t, "disks", argv[8])
}

func TestBuildArgs_FishWithHelp(t *testing.T) {
	// --help must not be interpreted by fish
	argv := BuildArgs("fish", Fish, "talosctl", []string{"--help"})

	assert.Equal(t, "--", argv[4], "fish must have -- before --help")
	assert.Equal(t, "--help", argv[5])
}

func TestBuildArgs_FishNoArgs(t *testing.T) {
	// No args = no -- needed
	argv := BuildArgs("fish", Fish, "talosctl", []string{})

	assert.Equal(t, "fish", argv[0])
	assert.Equal(t, "--no-config", argv[1])
	assert.Equal(t, "-c", argv[2])
	assert.Equal(t, "talosctl", argv[3])
	assert.Len(t, argv, 4, "should not have -- when no args")
}

func TestBuildArgs_BashWithFlags(t *testing.T) {
	// Bash uses shell name as $0 separator, not --
	argv := BuildArgs("bash", Bash, "talosctl", []string{"-n", "10.0.0.1"})

	assert.Equal(t, "bash", argv[0])
	assert.Equal(t, "--norc", argv[1])
	assert.Equal(t, "--noprofile", argv[2])
	assert.Equal(t, "-c", argv[3])
	assert.Contains(t, argv[4], `"$@"`)
	assert.Equal(t, "bash", argv[5], "bash uses shell name as $0 separator")
	assert.Equal(t, "-n", argv[6])
}

func TestBuildArgs_ZshWithFlags(t *testing.T) {
	argv := BuildArgs("zsh", Zsh, "talosctl", []string{"-n", "10.0.0.1"})

	assert.Equal(t, "zsh", argv[0])
	assert.Equal(t, "--no-rcs", argv[1])
	assert.Equal(t, "-c", argv[2])
	assert.Contains(t, argv[3], `"$@"`)
	assert.Equal(t, "zsh", argv[4], "zsh uses shell name as $0 separator")
	assert.Equal(t, "-n", argv[5])
}

func TestBuildArgs_UnknownShellDefaults(t *testing.T) {
	// Unknown shell type: no optimization flags, bash-style arg syntax
	argv := BuildArgs("sh", "sh", "echo hi", []string{"x"})

	assert.Equal(t, "sh", argv[0])
	assert.Equal(t, "-c", argv[1])
	assert.Contains(t, argv[2], `"$@"`)
	assert.Equal(t, "sh", argv[3])
	assert.Equal(t, "x", argv[4])
}

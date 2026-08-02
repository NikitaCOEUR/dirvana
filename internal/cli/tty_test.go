package cli

import (
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTTY substitutes the terminal with a socket pair, seeded with what
// the user is supposed to type. It returns a function reading back
// everything the code wrote to its side.
//
// A real /dev/tty cannot be used here: a test binary has no controlling
// terminal, which is precisely why the fallback exists. A regular file
// would not do either, since reads and writes would share one offset,
// whereas a terminal carries both directions independently.
func stubTTY(t *testing.T, input string) func() string {
	t.Helper()

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.NoError(t, err)

	ttySide := os.NewFile(uintptr(fds[0]), "tty")
	testSide := os.NewFile(uintptr(fds[1]), "tty-peer")
	t.Cleanup(func() {
		_ = ttySide.Close()
		_ = testSide.Close()
	})

	if input != "" {
		_, err = testSide.WriteString(input)
		require.NoError(t, err)
	}
	// Close this direction only: the code reads the seeded input then sees
	// end of input, exactly like a user pressing Enter then Ctrl-D, while
	// the other direction stays open to collect what it writes back
	require.NoError(t, syscall.Shutdown(fds[1], syscall.SHUT_WR))

	orig := openTTY
	openTTY = func(int) (*os.File, error) { return ttySide, nil }
	t.Cleanup(func() { openTTY = orig })

	return func() string {
		// The code under test closes its side, which ends this read
		data, _ := io.ReadAll(testSide)
		return string(data)
	}
}

// failingTTY makes every terminal open fail, as on a process with no
// controlling terminal
func failingTTY(t *testing.T) {
	t.Helper()
	orig := openTTY
	openTTY = func(int) (*os.File, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { openTTY = orig })
}

func TestNotifyUser_WritesToTerminal(t *testing.T) {
	readBack := stubTTY(t, "")

	out := captureStderr(t, func() {
		notifyUser("hello user\n")
	})

	assert.Equal(t, "hello user\n", readBack(), "the notice belongs on the terminal")
	assert.Empty(t, out, "stderr is discarded by the shell hook, nothing must go there")
}

func TestNotifyUser_FallsBackToStderr(t *testing.T) {
	failingTTY(t)

	out := captureStderr(t, func() {
		notifyUser("hello user\n")
	})
	assert.Equal(t, "hello user\n", out)
}

func TestNotifyUser_TestModeFallsBackToStderr(t *testing.T) {
	// The shell integration tests drive the real binary and rely on this
	// escape hatch to read the prompt from stdin
	t.Setenv("DIRVANA_TEST_MODE", "1")

	out := captureStderr(t, func() {
		notifyUser("hello user\n")
	})
	assert.Equal(t, "hello user\n", out)
}

func TestPromptShellApproval_OnTerminal(t *testing.T) {
	tests := []struct {
		answer   string
		approved bool
	}{
		{answer: "y\n", approved: true},
		{answer: "yes\n", approved: true},
		{answer: "n\n", approved: false},
		{answer: "\n", approved: false},
	}

	for _, tt := range tests {
		t.Run("answer "+tt.answer, func(t *testing.T) {
			readBack := stubTTY(t, tt.answer)

			out := captureStderr(t, func() {
				approved, err := promptShellApproval()
				require.NoError(t, err)
				assert.Equal(t, tt.approved, approved)
			})

			assert.Contains(t, readBack(), "Approve execution?", "the prompt belongs on the terminal")
			assert.Empty(t, out)
		})
	}
}

func TestPromptShellApproval_TerminalClosesWithoutAnswer(t *testing.T) {
	stubTTY(t, "")

	approved, err := promptShellApproval()
	require.Error(t, err)
	assert.False(t, approved)
}

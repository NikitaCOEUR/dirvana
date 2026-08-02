package cli

import (
	"errors"
	"fmt"
	"os"
)

// openTTY opens the user's terminal, which is where the interactive
// prompts and notices belong: the shell hook captures stdout and discards
// stderr (shell_code=$(dirvana export ... 2>/dev/null)), so anything
// written there would never reach the user.
//
// It is a variable so tests can substitute a file they control. /dev/tty
// is unavailable to a process without a controlling terminal, which is
// exactly the case under `go test` and in CI.
//
// DIRVANA_TEST_MODE forces the fallback path, so the shell integration
// tests driving the real binary can answer the prompt through stdin.
var openTTY = func(flag int) (*os.File, error) {
	if os.Getenv("DIRVANA_TEST_MODE") != "" {
		return nil, errors.New("terminal disabled by DIRVANA_TEST_MODE")
	}
	return os.OpenFile("/dev/tty", flag, 0)
}

// notifyUser writes a message to the user's terminal, falling back to
// stderr when no terminal can be reached
func notifyUser(msg string) {
	if tty, err := openTTY(os.O_WRONLY); err == nil {
		defer func() { _ = tty.Close() }()
		_, _ = fmt.Fprint(tty, msg)
		return
	}
	_, _ = fmt.Fprint(os.Stderr, msg)
}

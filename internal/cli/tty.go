package cli

import (
	"fmt"
	"os"
)

// notifyUser writes a message directly to the user's terminal so it stays
// visible even when the shell hook captures stdout and discards stderr
// (shell_code=$(dirvana export ... 2>/dev/null)).
// DIRVANA_TEST_MODE forces the stderr fallback for deterministic tests.
func notifyUser(msg string) {
	if os.Getenv("DIRVANA_TEST_MODE") == "" {
		if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
			defer func() { _ = tty.Close() }()
			_, _ = fmt.Fprint(tty, msg)
			return
		}
	}
	_, _ = fmt.Fprint(os.Stderr, msg)
}

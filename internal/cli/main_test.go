package cli

import (
	"os"
	"testing"
)

// TestMain forces the test-mode prompt fallback (stderr + stdin instead of
// /dev/tty) for the whole package: several tests exercise commands that may
// ask for shell-command consent, and opening /dev/tty under 'go test' in a
// real terminal would block waiting for input.
func TestMain(m *testing.M) {
	if os.Getenv("DIRVANA_TEST_MODE") == "" {
		_ = os.Setenv("DIRVANA_TEST_MODE", "1")
	}
	os.Exit(m.Run())
}

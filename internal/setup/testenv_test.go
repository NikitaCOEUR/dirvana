package setup

import (
	"testing"
)

// testHome points HOME at a scratch directory for the duration of the test,
// so hook installation never touches the real user configuration
func testHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

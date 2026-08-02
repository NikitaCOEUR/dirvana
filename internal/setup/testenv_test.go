package setup

import (
	"testing"
)

// testHome points HOME at a scratch directory for the duration of the test
// and returns it, so hook installation never touches the real user
// configuration
func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

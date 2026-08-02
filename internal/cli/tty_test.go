package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderr runs fn with os.Stderr redirected to a pipe and returns
// what was written
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	require.NoError(t, w.Close())
	buf := make([]byte, 16*1024)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestNotifyUser_TestModeFallsBackToStderr(t *testing.T) {
	t.Setenv("DIRVANA_TEST_MODE", "1")

	out := captureStderr(t, func() {
		notifyUser("hello user\n")
	})
	assert.Equal(t, "hello user\n", out)
}

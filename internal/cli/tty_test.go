package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifyUser_TestModeFallsBackToStderr(t *testing.T) {
	t.Setenv("DIRVANA_TEST_MODE", "1")

	out := captureStderr(t, func() {
		notifyUser("hello user\n")
	})
	assert.Equal(t, "hello user\n", out)
}

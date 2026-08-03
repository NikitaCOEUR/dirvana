package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBinaryDoesNotQueryTheTerminalAtStartup guards the whole binary against
// dependencies that talk to the terminal from an init().
//
// Bubble Tea v1 does exactly that: its init() calls lipgloss.HasDarkBackground,
// which writes an OSC 11 query and waits for the terminal to answer. Every
// dirvana command paid for it - including `dirvana exec`, behind every alias,
// and anything the shell hook runs on cd. On a terminal that does not answer
// the query that is a five second stall, and the escape sequences show up on
// the user's prompt.
//
// Listing dependencies is coarse but cheap and stable; the alternative is a
// pty harness for a property that is really about what gets linked in.
func TestBinaryDoesNotQueryTheTerminalAtStartup(t *testing.T) {
	forbidden := map[string]string{
		"github.com/charmbracelet/bubbletea": "queries the terminal from init(); use charm.land/bubbletea/v2",
	}

	out, err := exec.Command("go", "list", "-deps", ".").Output()
	require.NoError(t, err, "go list failed")

	for _, dep := range strings.Split(string(out), "\n") {
		reason, banned := forbidden[strings.TrimSpace(dep)]
		require.False(t, banned, "%s must not be linked into dirvana: %s", dep, reason)
	}
}

package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBinaryStaysCheapToStart guards what every dirvana command pays before it
// does anything: package init.
//
// dirvana runs on every cd and behind every alias, so a dependency that builds
// tables at init is charged to all of them. Two have been paid for already:
//
//   - Bubble Tea v1 calls lipgloss.HasDarkBackground from an init(), which
//     queries the terminal and waits for an answer - five seconds of stall on
//     a terminal that never sends one.
//   - go-runewidth fills 2.2MB of Unicode lookup tables at init, 24ms on every
//     single command, for a `dirvana export` that measures nothing.
//
// Listing dependencies is coarse but cheap and stable; the alternative is a
// pty harness for a property that is really about what gets linked in.
func TestBinaryStaysCheapToStart(t *testing.T) {
	forbidden := map[string]string{
		"github.com/charmbracelet/bubbletea": "queries the terminal from init()",
		"github.com/mattn/go-runewidth":      "builds 2.2MB of lookup tables at init; use clipperhouse/displaywidth",
		"github.com/charmbracelet/lipgloss":  "pulls in go-runewidth through x/ansi; use internal/tui",
		"charm.land/bubbletea/v2":            "pulls in go-runewidth through x/ansi; use internal/tui",
	}

	out, err := exec.Command("go", "list", "-deps", ".").Output()
	require.NoError(t, err, "go list failed")

	for _, dep := range strings.Split(string(out), "\n") {
		reason, banned := forbidden[strings.TrimSpace(dep)]
		require.False(t, banned, "%s must not be linked into dirvana: %s", dep, reason)
	}
}

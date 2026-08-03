package cli

import (
	"fmt"
	"os"

	"github.com/NikitaCOEUR/dirvana/internal/status"
	"github.com/charmbracelet/x/term"
)

// StatusParams contains parameters for the Status command
type StatusParams struct {
	CachePath string
	AuthPath  string
	// Plain forces the scrolling output even on a terminal.
	Plain bool
}

// isTerminal reports whether a file is attached to a terminal rather than a
// pipe, a file or /dev/null. It is a variable so tests can decide which path
// they take.
//
// A mode check is not enough here: os.ModeCharDevice is set for /dev/null too,
// so `dirvana status > /dev/null` would take over the terminal and draw into
// the void until interrupted.
var isTerminal = func(f *os.File) bool {
	return term.IsTerminal(f.Fd())
}

// runInteractive is a variable for the same reason as isTerminal: a test can
// exercise the branch without a program taking over the terminal.
var runInteractive = status.RunInteractive

// Status displays the current Dirvana configuration status. On a terminal it
// opens the foldable view; anywhere else - a pipe, a log, `--plain` - it
// prints everything at once.
func Status(params StatusParams) error {
	data, err := status.CollectAll(params.CachePath, params.AuthPath)
	if err != nil {
		return fmt.Errorf("failed to collect status data: %w", err)
	}

	if params.Plain || !isTerminal(os.Stdout) || !isTerminal(os.Stdin) {
		fmt.Println(status.Render(data))
		return nil
	}

	return runInteractive(data, os.Stdin, os.Stdout)
}

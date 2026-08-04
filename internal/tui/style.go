// Package tui draws terminal output: styled text, aligned columns and a
// foldable full-screen view.
//
// It exists instead of a styling and TUI framework because dirvana runs on
// every cd and behind every alias, and the frameworks it replaces cost 27ms of
// package init on each of those - building 2.2MB of Unicode lookup tables in
// go-runewidth that a `dirvana export` never touches. What was actually used
// of them fits here: SGR colours, display width, a rounded box, and a key
// loop.
package tui

import (
	"os"
	"strconv"
	"strings"

	"github.com/clipperhouse/displaywidth"
	"golang.org/x/term"
)

// ansiAware measures text as a terminal would, skipping escape sequences
// rather than counting them as characters.
var ansiAware = displaywidth.Options{ControlSequences: true}

// Style is a set of SGR attributes to wrap text in. The zero value renders
// text unchanged.
type Style struct {
	codes []string
}

// NewStyle returns a style that changes nothing yet.
func NewStyle() Style { return Style{} }

// Bold returns the style with bold added.
func (s Style) Bold() Style {
	return s.with("1")
}

// Foreground returns the style painted in one of the 256 terminal colours.
func (s Style) Foreground(color int) Style {
	return s.with("38;5;" + strconv.Itoa(color))
}

func (s Style) with(code string) Style {
	// Copy: a style handed out must not change under its holder
	codes := make([]string, len(s.codes), len(s.codes)+1)
	copy(codes, s.codes)
	return Style{codes: append(codes, code)}
}

// Render wraps text in the style's escape sequences, or returns it untouched
// when colour is off or the style is empty.
func (s Style) Render(text string) string {
	if len(s.codes) == 0 || text == "" || !ColorEnabled() {
		return text
	}
	return "\x1b[" + strings.Join(s.codes, ";") + "m" + text + "\x1b[0m"
}

// colorEnabled is resolved once: it depends on the environment and on where
// output goes, neither of which changes while a command runs.
var colorEnabled = resolveColor()

// ColorEnabled reports whether escape sequences should be emitted at all.
func ColorEnabled() bool { return colorEnabled }

// SetColorEnabled forces the answer, for tests and for callers that know
// better than the environment.
func SetColorEnabled(enabled bool) { colorEnabled = enabled }

// resolveColor follows the conventions every terminal program is expected to:
// NO_COLOR wins over everything, a dumb terminal gets nothing, and output that
// is not a terminal gets plain text so that a pipe stays parseable.
func resolveColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Width returns how many terminal cells a string occupies, escape sequences
// excluded.
func Width(s string) int {
	return ansiAware.String(s)
}

// Height returns how many lines a string occupies.
func Height(s string) int {
	return strings.Count(s, "\n") + 1
}

// Truncate shortens text to width cells, escape sequences excluded, appending
// tail when it had to cut. It never splits an escape sequence.
func Truncate(s string, width int, tail string) string {
	if width <= 0 {
		return ""
	}
	return ansiAware.TruncateString(s, width, tail)
}

// Pad extends text with spaces to width cells. Text already that wide is
// returned unchanged.
func Pad(s string, width int) string {
	if gap := width - Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

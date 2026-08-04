package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withColor turns colour on for one test and puts it back afterwards, since
// the answer is resolved once per process from the environment.
func withColor(t *testing.T, enabled bool) {
	t.Helper()
	previous := ColorEnabled()
	SetColorEnabled(enabled)
	t.Cleanup(func() { SetColorEnabled(previous) })
}

func TestStyle_Render(t *testing.T) {
	withColor(t, true)

	assert.Equal(t, "\x1b[1mbold\x1b[0m", NewStyle().Bold().Render("bold"))
	assert.Equal(t, "\x1b[38;5;12mblue\x1b[0m", NewStyle().Foreground(12).Render("blue"))
	assert.Equal(t, "\x1b[1;38;5;9mboth\x1b[0m", NewStyle().Bold().Foreground(9).Render("both"))

	// Nothing to say, nothing added
	assert.Equal(t, "plain", NewStyle().Render("plain"))
	assert.Empty(t, NewStyle().Bold().Render(""))
}

func TestStyle_RenderWithoutColor(t *testing.T) {
	withColor(t, false)

	// A pipe, a log or NO_COLOR gets text that stays parseable
	assert.Equal(t, "bold", NewStyle().Bold().Foreground(12).Render("bold"))
}

// TestStyle_IsImmutable covers the trap of a shared base style: deriving from
// one must not change it for everyone else holding it.
func TestStyle_IsImmutable(t *testing.T) {
	withColor(t, true)

	base := NewStyle().Foreground(12)
	derived := base.Bold()

	assert.Equal(t, "\x1b[38;5;12mx\x1b[0m", base.Render("x"))
	assert.Equal(t, "\x1b[38;5;12;1mx\x1b[0m", derived.Render("x"))
}

func TestWidth(t *testing.T) {
	withColor(t, true)

	assert.Equal(t, 5, Width("hello"))
	assert.Equal(t, 4, Width("élan"))
	assert.Equal(t, 6, Width("日本語"), "east asian characters take two cells each")
	assert.Equal(t, 2, Width("📂"), "so do emoji")

	// Escape sequences occupy no cells on screen
	assert.Equal(t, 5, Width(NewStyle().Bold().Foreground(12).Render("hello")))
}

func TestHeight(t *testing.T) {
	assert.Equal(t, 1, Height("one line"))
	assert.Equal(t, 2, Height("two\nlines"))
	assert.Equal(t, 1, Height(""))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", Truncate("hello", 10, "…"))
	assert.Equal(t, "hel…", Truncate("hello", 4, "…"))
	assert.Equal(t, "élan…", Truncate("élan vital", 5, "…"))
	assert.Empty(t, Truncate("hello", 0, "…"))
	assert.Empty(t, Truncate("hello", -1, "…"))
}

// TestTruncate_KeepsEscapeSequencesIntact is why this goes through a
// width-aware truncation rather than slicing runes: a cut landing inside an
// escape sequence leaves the terminal painting the rest of the screen in that
// colour.
func TestTruncate_KeepsEscapeSequencesIntact(t *testing.T) {
	withColor(t, true)

	styled := NewStyle().Bold().Foreground(12).Render("hello world")
	cut := Truncate(styled, 6, "…")

	assert.LessOrEqual(t, Width(cut), 6)
	assert.True(t, strings.HasSuffix(cut, "\x1b[0m"), "the reset must survive: %q", cut)
	assert.Contains(t, cut, "…")
}

func TestPad(t *testing.T) {
	assert.Equal(t, "ab   ", Pad("ab", 5))
	assert.Equal(t, "abcde", Pad("abcde", 5))
	assert.Equal(t, "abcdef", Pad("abcdef", 5), "text wider than the column is left alone")

	// Padding is measured in cells, not bytes
	assert.Equal(t, 5, Width(Pad("日本", 5)))
}

func TestResolveColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	assert.False(t, resolveColor(), "NO_COLOR wins over everything")

	t.Setenv("NO_COLOR", "")
	assert.False(t, resolveColor(), "even empty: the variable being set is the signal")

	require.NoError(t, os.Unsetenv("NO_COLOR"))
	t.Setenv("TERM", "dumb")
	assert.False(t, resolveColor())
}

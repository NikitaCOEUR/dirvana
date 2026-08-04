package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"a letter", "q", []string{"q"}},
		{"several letters in one burst", "jjk", []string{"j", "j", "k"}},
		{"enter", "\r", []string{"enter"}},
		{"newline is enter too", "\n", []string{"enter"}},
		{"tab", "\t", []string{"tab"}},
		{"space", " ", []string{"space"}},
		{"ctrl+c", "\x03", []string{"ctrl+c"}},
		{"ctrl+d", "\x04", []string{"ctrl+d"}},
		{"backspace", "\x7f", []string{"backspace"}},
		{"a lone escape", "\x1b", []string{"esc"}},

		// Arrows arrive as a burst holding nothing else, which is the case a
		// length check must not give up on
		{"up", "\x1b[A", []string{"up"}},
		{"down", "\x1b[B", []string{"down"}},
		{"right", "\x1b[C", []string{"right"}},
		{"left", "\x1b[D", []string{"left"}},
		{"home", "\x1b[H", []string{"home"}},
		{"end", "\x1b[F", []string{"end"}},

		// Application cursor mode, which some terminals switch to
		{"up in SS3", "\x1bOA", []string{"up"}},
		{"end in SS3", "\x1bOF", []string{"end"}},

		// Four bytes, and the prefix "[5" must not win over "[5~"
		{"page up", "\x1b[5~", []string{"pgup"}},
		{"page down", "\x1b[6~", []string{"pgdown"}},

		{"a sequence then a letter", "\x1b[Aq", []string{"up", "q"}},
		{"two sequences", "\x1b[A\x1b[B", []string{"up", "down"}},
		{"an unknown sequence degrades to esc", "\x1b[Z", []string{"esc", "[", "Z"}},

		{"a multi-byte rune", "é", []string{"é"}},
		{"an emoji", "📂", []string{"📂"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, decodeKeys([]byte(tt.input)))
		})
	}
}

// countingApp records the keys it was given and stops on "q".
type countingApp struct {
	keys          []string
	width, height int
	frames        int
}

func (a *countingApp) Key(key string) bool {
	a.keys = append(a.keys, key)
	return key != "q"
}

func (a *countingApp) Resize(width, height int) { a.width, a.height = width, height }

func (a *countingApp) View() string {
	a.frames++
	return "frame"
}

func TestRun_DeliversKeysAndStops(t *testing.T) {
	app := &countingApp{}

	in, keyboard, err := os.Pipe()
	require.NoError(t, err)
	go func() {
		_, _ = keyboard.WriteString("j\x1b[Bq")
		_ = keyboard.Close()
	}()

	var out strings.Builder
	require.NoError(t, Run(app, in, &out))

	assert.Equal(t, []string{"j", "down", "q"}, app.keys)
	assert.Contains(t, out.String(), enterAltScreen, "the terminal must be left as it was found")
	assert.Contains(t, out.String(), leaveAltScreen)
	assert.Contains(t, out.String(), "frame")
}

func TestRun_StopsWhenInputEnds(t *testing.T) {
	app := &countingApp{}

	in, keyboard, err := os.Pipe()
	require.NoError(t, err)
	_ = keyboard.Close() // nothing to read, ever

	var out strings.Builder
	require.NoError(t, Run(app, in, &out))

	assert.Empty(t, app.keys)
	assert.Positive(t, app.frames, "the first frame is drawn before any key")
}

func TestRun_NewlinesBecomeCarriageReturns(t *testing.T) {
	// A raw terminal does not move to column 0 on its own, so a view written
	// with plain newlines would stair-step down the screen
	app := &staticApp{view: "one\ntwo"}

	in, keyboard, err := os.Pipe()
	require.NoError(t, err)
	_ = keyboard.Close()

	var out strings.Builder
	require.NoError(t, Run(app, in, &out))

	assert.Contains(t, out.String(), "one\r\ntwo")
}

type staticApp struct{ view string }

func (a *staticApp) Key(string) bool { return false }
func (a *staticApp) Resize(_, _ int) {}
func (a *staticApp) View() string    { return a.view }

func TestReadKeys_ClosesOnEOF(t *testing.T) {
	keys := readKeys(strings.NewReader("ab"))

	var got []string
	for key := range keys {
		got = append(got, key)
	}

	assert.Equal(t, []string{"a", "b"}, got)
}

func TestDecodeKeys_InvalidUTF8(t *testing.T) {
	// A stray byte must advance the loop rather than spin on it
	keys := decodeKeys([]byte{0xff, 'q'})

	assert.Contains(t, keys, "q", "what follows a bad byte is still read")
	assert.Len(t, keys, 2)
}

func TestDecodeKeys_EscapeAtTheEndOfABurst(t *testing.T) {
	// Not enough bytes left to be a sequence: it is the Escape key
	assert.Equal(t, []string{"a", "esc"}, decodeKeys([]byte("a\x1b")))
	assert.Equal(t, []string{"a", "esc", "["}, decodeKeys([]byte("a\x1b[")))
}

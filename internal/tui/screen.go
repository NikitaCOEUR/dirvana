package tui

import (
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unicode/utf8"

	"golang.org/x/term"
)

// App is a full-screen view driven by keystrokes.
type App interface {
	// Key handles a keystroke by name - "up", "enter", "q", "ctrl+c" - and
	// reports whether the view should keep running.
	Key(key string) bool
	// Resize records a new terminal size.
	Resize(width, height int)
	// View renders the whole screen.
	View() string
}

// Screen control sequences. The alternate screen is what lets the view leave
// the terminal exactly as it found it.
const (
	enterAltScreen = "\x1b[?1049h"
	leaveAltScreen = "\x1b[?1049l"
	hideCursor     = "\x1b[?25l"
	showCursor     = "\x1b[?25h"
	clearScreen    = "\x1b[H\x1b[2J"
)

// Run drives an App until it asks to stop, or until input runs out.
//
// The terminal is put in raw mode so keystrokes arrive as they are typed, and
// restored on the way out - including when the app panics, since a terminal
// left raw is unusable.
func Run(app App, in *os.File, out io.Writer) (err error) {
	if width, height, sizeErr := term.GetSize(int(in.Fd())); sizeErr == nil {
		app.Resize(width, height)
	}

	if term.IsTerminal(int(in.Fd())) {
		state, rawErr := term.MakeRaw(int(in.Fd()))
		if rawErr != nil {
			return rawErr
		}
		defer func() {
			if restoreErr := term.Restore(int(in.Fd()), state); err == nil {
				err = restoreErr
			}
		}()
	}

	_, _ = io.WriteString(out, enterAltScreen+hideCursor)
	defer func() { _, _ = io.WriteString(out, showCursor+leaveAltScreen) }()

	resized := make(chan os.Signal, 1)
	signal.Notify(resized, syscall.SIGWINCH)
	defer signal.Stop(resized)

	keys := readKeys(in)
	draw(app, out)

	for {
		select {
		case key, ok := <-keys:
			if !ok {
				return err
			}
			if !app.Key(key) {
				return err
			}
		case <-resized:
			if width, height, sizeErr := term.GetSize(int(in.Fd())); sizeErr == nil {
				app.Resize(width, height)
			}
		}
		draw(app, out)
	}
}

func draw(app App, out io.Writer) {
	// One write per frame: two would let the terminal paint a half-drawn
	// screen between them
	_, _ = io.WriteString(out, clearScreen+strings.ReplaceAll(app.View(), "\n", "\r\n"))
}

// readKeys decodes keystrokes on its own goroutine and closes the channel when
// input ends, which is how a piped session stops.
func readKeys(in io.Reader) <-chan string {
	keys := make(chan string)

	go func() {
		defer close(keys)

		buf := make([]byte, 64)
		for {
			n, err := in.Read(buf)
			if n > 0 {
				for _, key := range decodeKeys(buf[:n]) {
					keys <- key
				}
			}
			if err != nil {
				return
			}
		}
	}()

	return keys
}

// escapeSequences maps what follows a CSI introducer to a key name. A terminal
// sends these as one burst, so they arrive in the same read as the escape that
// introduces them.
var escapeSequences = map[string]string{
	"[A": "up", "[B": "down", "[C": "right", "[D": "left",
	"[H": "home", "[F": "end",
	"[5~": "pgup", "[6~": "pgdown",
	"OA": "up", "OB": "down", "OC": "right", "OD": "left",
	"OH": "home", "OF": "end",
}

// decodeKeys turns a burst of input bytes into key names.
func decodeKeys(input []byte) []string {
	var keys []string

	for i := 0; i < len(input); {
		b := input[i]

		if b == 0x1b {
			if key, size := decodeEscape(input[i:]); key != "" {
				keys = append(keys, key)
				i += size
				continue
			}
			// A lone escape, or one this does not know: treat it as Esc so
			// the view can still be left with a single keypress
			keys = append(keys, "esc")
			i++
			continue
		}

		key, size := decodeRune(input[i:])
		keys = append(keys, key)
		i += size
	}

	return keys
}

// decodeEscape recognises a CSI or SS3 sequence, returning how many bytes it
// consumed.
func decodeEscape(input []byte) (key string, size int) {
	// Longest first, so "[5~" is not mistaken for an unknown "[5". A length
	// that runs past the input is skipped, not a reason to give up: the burst
	// may hold exactly one sequence and nothing more.
	for length := 3; length >= 2; length-- {
		if 1+length > len(input) {
			continue
		}
		if key, ok := escapeSequences[string(input[1:1+length])]; ok {
			return key, 1 + length
		}
	}
	return "", 0
}

// decodeRune names a single keystroke, resolving control characters to the
// names the rest of the program matches on.
func decodeRune(input []byte) (key string, size int) {
	switch b := input[0]; {
	case b == '\r' || b == '\n':
		return "enter", 1
	case b == '\t':
		return "tab", 1
	case b == 0x7f || b == 0x08:
		return "backspace", 1
	case b == ' ':
		return "space", 1
	case b < 0x20:
		// Control characters are the letter they are typed with: 0x03 is
		// ctrl+c, which is 'c' at 0x63 minus 0x60
		return "ctrl+" + string(rune(b+0x60)), 1
	}

	// Anything else is text, and may be multi-byte. DecodeRune reports how
	// much of the input the rune actually took, which for a byte that is not
	// valid UTF-8 is one - converting the whole buffer instead would swallow
	// whatever was typed after it.
	r, size := utf8.DecodeRune(input)
	return string(r), size
}

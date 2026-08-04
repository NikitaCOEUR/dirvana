package status

import (
	"fmt"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/tui"
)

const (
	// rowIndent is the left margin of every row under a section title.
	rowIndent = 3
	// detailIndent lines continuation rows up past their parent's key.
	detailIndent = 6
	// colGap separates the key, value and note columns.
	colGap = 2
)

var (
	borderStyle  = tui.NewStyle().Foreground(240)
	versionStyle = tui.NewStyle().Bold().Foreground(12)
	dirStyle     = tui.NewStyle().Foreground(15)
	sectionStyle = tui.NewStyle().Bold().Foreground(14)
	keyStyle     = tui.NewStyle().Foreground(15)
	valueStyle   = tui.NewStyle().Foreground(252)

	toneStyles = map[Tone]tui.Style{
		ToneNeutral: tui.NewStyle(),
		ToneMuted:   tui.NewStyle().Foreground(244),
		ToneOK:      tui.NewStyle().Foreground(10),
		ToneWarn:    tui.NewStyle().Foreground(11),
		ToneError:   tui.NewStyle().Foreground(9),
	}
)

// Render writes the whole status as plain scrolling output, every section
// unfolded. This is what non-interactive callers get: a pipe, a CI log, or
// `dirvana status --plain`.
func Render(data *Data) string {
	var b strings.Builder

	b.WriteString(RenderHeader(BuildHeader(data)))
	b.WriteString("\n")

	for _, section := range BuildSections(data) {
		b.WriteString("\n")
		b.WriteString(RenderSectionTitle(section, ""))
		b.WriteString("\n")
		for _, line := range RenderRows(section.Rows, 0) {
			b.WriteString(line + "\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// RenderHeader frames what dirvana is and where it is looking, over the state
// worth knowing before reading any section.
func RenderHeader(h Header) string {
	var b strings.Builder
	b.WriteString(versionStyle.Render("dirvana " + h.Version))
	b.WriteString("  ")
	b.WriteString(dirStyle.Render(h.Dir))

	if len(h.Chips) > 0 {
		parts := make([]string, 0, len(h.Chips))
		for _, chip := range h.Chips {
			parts = append(parts, toneStyles[chip.Tone].Render(chip.Text))
		}
		b.WriteString("\n")
		b.WriteString(strings.Join(parts, toneStyles[ToneMuted].Render(" · ")))
	}

	return box(b.String())
}

// RenderSectionTitle renders a section's heading. marker is prepended as-is,
// which is how the interactive view shows whether a section is folded.
func RenderSectionTitle(section Section, marker string) string {
	title := sectionStyle.Render(section.Icon + "  " + section.Title)
	if section.Summary != "" {
		title += "  " + toneStyles[ToneMuted].Render(section.Summary)
	}
	return marker + title
}

// RenderRows lays rows out in aligned columns: keys padded to a common width,
// notes pinned to a common column past the longest value.
//
// width is the space available; pass 0 to let the rows size themselves, which
// keeps the output identical whatever terminal produced it - what a pipe or a
// test wants. A positive width shortens values that would not fit.
func RenderRows(rows []Row, width int) []string {
	l := layout{width: width, noteWidth: widestNote(rows)}
	l.keyWidth, l.noteColumn = measureRows(rows)

	if width > 0 {
		// Notes carry the actionable half of a row - "not approved", the
		// completion a command resolves to - so they are the last to give way
		l.noteWidth = min(l.noteWidth, max(width-rowIndent, 0))

		// Then the key column, so that a long key cannot push the line past
		// the right edge on its own
		if l.keyWidth > 0 {
			l.keyWidth = max(min(l.keyWidth, width-rowIndent-colGap-l.noteWidth), 1)
		}

		// Pull the notes in when the terminal cannot hold the natural layout
		l.noteColumn = min(l.noteColumn, width-l.noteWidth)
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, renderRow(row, l))
	}
	return lines
}

// layout is the column geometry shared by every row of a section.
type layout struct {
	keyWidth   int
	noteColumn int
	// noteWidth is what the note column takes, its leading gap included, and
	// 0 when no row carries a note.
	noteWidth int
	// width is the space available, or 0 to let the rows size themselves.
	width int
}

// renderRow assembles one line. Widths are measured on the plain text and the
// styles applied per segment, so a truncation never lands inside an escape
// sequence.
func renderRow(row Row, l layout) string {
	tone := toneStyles[row.Tone]

	if row.Detail {
		value := row.Value
		if l.width > 0 {
			value = truncate(value, l.width-detailIndent)
		}
		return strings.Repeat(" ", detailIndent) + tone.Render(value)
	}

	var b strings.Builder
	col := rowIndent
	b.WriteString(strings.Repeat(" ", rowIndent))

	if l.keyWidth > 0 {
		key := truncate(row.Key, l.keyWidth)
		// Nothing follows, so nothing to line up against
		if row.Value == "" && row.Note == "" {
			return b.String() + keyStyle.Render(key)
		}
		b.WriteString(keyStyle.Render(pad(key, l.keyWidth)))
		b.WriteString(strings.Repeat(" ", colGap))
		col += l.keyWidth + colGap
	}

	note := row.Note
	if l.width > 0 && note != "" {
		note = truncate(note, l.noteWidth-colGap)
	}

	// How much room the value has: it must clear the note column, and it must
	// clear the right edge. Both constraints apply - honouring only the note
	// column let a long key push the line past the terminal width.
	limited, room := false, 0
	if note != "" && l.noteColumn > 0 {
		limited, room = true, l.noteColumn-col-colGap
	}
	if l.width > 0 {
		if edge := l.width - col - widthOf(note, colGap); !limited || edge < room {
			limited, room = true, edge
		}
	}
	value := row.Value
	if limited {
		value = truncate(value, max(room, 0))
	}

	// The value carries the row's tone when there is no note to carry it
	if note == "" && row.Tone != ToneNeutral {
		b.WriteString(tone.Render(value))
	} else {
		b.WriteString(valueStyle.Render(value))
	}
	col += tui.Width(value)

	if note != "" {
		padding := max(l.noteColumn-col, colGap)
		b.WriteString(strings.Repeat(" ", padding))
		b.WriteString(tone.Render(note))
	}

	line := b.String()
	if l.width > 0 {
		// Last resort: too narrow for any arrangement to fit. Cutting here is
		// ANSI-aware, so it can never land inside an escape sequence.
		line = tui.Truncate(line, l.width, "")
	}
	return line
}

// measureRows returns the width the key column needs and the column the notes
// should start at, so every row of a section lines up.
func measureRows(rows []Row) (keyWidth, noteColumn int) {
	for _, row := range rows {
		if !row.Detail {
			keyWidth = max(keyWidth, tui.Width(row.Key))
		}
	}

	for _, row := range rows {
		if row.Detail || row.Note == "" {
			continue
		}
		body := rowIndent + tui.Width(row.Value)
		if keyWidth > 0 {
			body += keyWidth + colGap
		}
		noteColumn = max(noteColumn, body+colGap)
	}
	return keyWidth, noteColumn
}

// widestNote returns the room the note column needs, gap included.
func widestNote(rows []Row) int {
	widest := 0
	for _, row := range rows {
		if !row.Detail {
			widest = max(widest, tui.Width(row.Note))
		}
	}
	if widest > 0 {
		widest += colGap
	}
	return widest
}

func pad(s string, width int) string {
	return tui.Pad(s, width)
}

// truncate shortens plain text to width display cells, marking the cut with an
// ellipsis. The result never exceeds width, ellipsis included.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if tui.Width(s) <= width {
		return s
	}
	// Measured once and walked once; dropping runes one at a time and
	// re-measuring is quadratic, and these strings are re-truncated on every
	// frame of the interactive view
	return strings.TrimRight(tui.Truncate(s, width-1, ""), " ") + "…"
}

// widthOf returns the space a piece of text needs including its leading gap,
// or nothing at all when there is no text.
func widthOf(s string, gap int) int {
	if s == "" {
		return 0
	}
	return tui.Width(s) + gap
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// box draws a rounded frame around a block of lines, sized to its widest one.
// It replaces what a styling framework did for this one header, and is the
// only border dirvana draws.
func box(content string) string {
	lines := strings.Split(content, "\n")
	inner := 0
	for _, line := range lines {
		inner = max(inner, tui.Width(line))
	}

	horizontal := strings.Repeat("─", inner+2)

	var b strings.Builder
	b.WriteString(borderStyle.Render("╭"+horizontal+"╮") + "\n")
	for _, line := range lines {
		b.WriteString(borderStyle.Render("│") + " " + pad(line, inner) + " " + borderStyle.Render("│") + "\n")
	}
	b.WriteString(borderStyle.Render("╰" + horizontal + "╯"))

	return b.String()
}

package status

import (
	"io"
	"strings"

	// bubbletea v2, not v1: v1 queries the terminal for its background colour
	// from an init(), which every dirvana command would pay for - including
	// the ones the shell hook runs on each cd.
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

// gutter is the plain left margin every line carries, and where the cursor is
// drawn. Plain matters: the cursor overwrites those columns, so no escape
// sequence may start there - a fold marker drawn in it would be eaten.
const gutter = 2

var (
	cursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	fullStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// line is one navigable entry of the view. A line is either a section heading,
// which folds, or one of its rows.
type line struct {
	section int
	row     int // -1 for the heading itself
}

// tuiModel drives the interactive status view.
type tuiModel struct {
	data     *Data
	header   Header
	sections []Section

	lines  []line
	cursor int
	offset int

	// notice reports the outcome of the last action taken from the view.
	notice     string
	noticeTone Tone

	width  int
	height int
}

// actionDone carries an action's outcome back into the view.
type actionDone struct {
	message string
	err     error
}

// RunInteractive shows the status as a foldable view and returns once the user
// leaves it. Callers are expected to have checked that they are on a terminal;
// Render is the answer when they are not.
func RunInteractive(data *Data, in io.Reader, out io.Writer) error {
	m := &tuiModel{
		data:     data,
		header:   BuildHeader(data),
		sections: BuildSections(data),
		width:    80,
		height:   24,
	}
	m.rebuild()

	_, err := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out)).Run()
	return err
}

// cursorTarget names the line the cursor is on by section title rather than by
// index, so it survives sections appearing or disappearing - Setup drops out
// once its fix has been applied, shifting every index below it.
type cursorTarget struct {
	title string
	row   int
}

func (m *tuiModel) cursorTarget() cursorTarget {
	if m.cursor >= len(m.lines) {
		return cursorTarget{row: -1}
	}
	current := m.lines[m.cursor]
	return cursorTarget{title: m.sections[current.section].Title, row: current.row}
}

// rebuild recomputes the navigable lines after a fold changed, keeping the
// cursor where it was.
func (m *tuiModel) rebuild() {
	m.rebuildTo(m.cursorTarget())
}

// rebuildTo recomputes the navigable lines and puts the cursor back on the
// given target. Callers that replace m.sections must capture the target first,
// while the old sections are still there to name it.
func (m *tuiModel) rebuildTo(target cursorTarget) {
	m.lines = m.lines[:0]
	for i, section := range m.sections {
		m.lines = append(m.lines, line{section: i, row: -1})
		if !section.Expanded {
			continue
		}
		for r := range section.Rows {
			m.lines = append(m.lines, line{section: i, row: r})
		}
	}

	m.cursor = 0
	for i, l := range m.lines {
		if m.sections[l.section].Title != target.title {
			continue
		}
		// Fall back to the heading: the row may have been folded away, or the
		// section may not have that many rows any more
		m.cursor = i
		if l.row == target.row {
			break
		}
	}
}

func (m *tuiModel) Init() tea.Cmd { return nil }

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case actionDone:
		m.applyActionResult(msg)
	case tea.KeyPressMsg:
		return m, m.handleKey(msg.String())
	}
	return m, nil
}

// applyActionResult reports what an action did and refreshes what it changed,
// so the header stops warning about a hook that is now installed.
func (m *tuiModel) applyActionResult(msg actionDone) {
	if msg.err != nil {
		m.notice, m.noticeTone = msg.err.Error(), ToneError
		return
	}
	m.notice, m.noticeTone = msg.message, ToneOK

	collectSystemInfo(m.data)
	m.header = BuildHeader(m.data)
	m.refreshSections()
}

// refreshSections rebuilds the sections while keeping every fold as the user
// left it. Sections that appear or disappear - Setup, once it is fixed - take
// their default state.
func (m *tuiModel) refreshSections() {
	// Both have to be read while the old sections are still in place
	target := m.cursorTarget()
	folds := make(map[string]bool, len(m.sections))
	for _, section := range m.sections {
		folds[section.Title] = section.Expanded
	}

	m.sections = BuildSections(m.data)
	for i, section := range m.sections {
		if expanded, ok := folds[section.Title]; ok {
			m.sections[i].Expanded = expanded
		}
	}
	m.rebuildTo(target)
}

// handleKey acts on a key by name, which is all the view needs to know about
// the terminal.
func (m *tuiModel) handleKey(key string) tea.Cmd {
	switch key {
	case "q", "esc", "ctrl+c":
		return tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-m.bodyHeight())
	case "pgdown":
		m.move(m.bodyHeight())
	case "home", "g":
		m.move(-len(m.lines))
	case "end", "G":
		m.move(len(m.lines))
	case "right", "l":
		m.setFold(true)
	case "left", "h":
		m.setFold(false)
	case "enter", "space", " ", "tab":
		// A row offering a fix is what enter is for; folding is what the rest
		// of the line does
		if action := m.currentAction(); action != nil {
			return m.run(action)
		}
		m.toggleFold()
	case "a":
		m.toggleAll()
	}
	return nil
}

// currentAction returns the fix offered by the selected row, if any.
func (m *tuiModel) currentAction() *Action {
	if m.cursor >= len(m.lines) {
		return nil
	}
	l := m.lines[m.cursor]
	if l.row < 0 {
		return nil
	}
	return m.sections[l.section].Rows[l.row].Action
}

// run performs an action off the event loop, so a slow install does not freeze
// the view.
func (m *tuiModel) run(action *Action) tea.Cmd {
	m.notice, m.noticeTone = "running: "+action.Label+"…", ToneMuted
	return func() tea.Msg {
		message, err := action.Run()
		return actionDone{message: message, err: err}
	}
}

func (m *tuiModel) move(delta int) {
	m.cursor = min(max(m.cursor+delta, 0), max(len(m.lines)-1, 0))
	// The footer belongs to the line under the cursor again
	m.notice = ""
}

// currentSection returns the section the cursor sits in, heading or row alike.
func (m *tuiModel) currentSection() int {
	if m.cursor >= len(m.lines) {
		return -1
	}
	return m.lines[m.cursor].section
}

func (m *tuiModel) setFold(expanded bool) {
	idx := m.currentSection()
	if idx < 0 || m.sections[idx].Expanded == expanded {
		return
	}
	m.sections[idx].Expanded = expanded
	m.rebuild()
}

func (m *tuiModel) toggleFold() {
	if idx := m.currentSection(); idx >= 0 {
		m.setFold(!m.sections[idx].Expanded)
	}
}

// toggleAll unfolds everything, or folds everything back once nothing is left
// to unfold.
func (m *tuiModel) toggleAll() {
	expand := false
	for _, section := range m.sections {
		if !section.Expanded {
			expand = true
			break
		}
	}
	for i := range m.sections {
		m.sections[i].Expanded = expand
	}
	m.rebuild()
}

// View hands the rendered screen to bubbletea, on the alternate screen so the
// terminal is left as it was found.
func (m *tuiModel) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true
	return view
}

func (m *tuiModel) render() string {
	header := RenderHeader(m.header)
	height := m.bodyHeightBelow(header)
	m.scrollTo(height)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(strings.Join(m.body(m.offset, min(m.offset+height, len(m.lines))), "\n"))
	b.WriteString("\n\n")
	b.WriteString(m.footer(height))
	return b.String()
}

// body renders the lines in [from, to), with the cursor drawn in the left
// margin the gutter reserves. Only what is on screen is formatted: the rest
// would be thrown away, and every keystroke re-renders.
func (m *tuiModel) body(from, to int) []string {
	rendered := make([]string, 0, to-from)
	rowLines := make(map[int][]string, len(m.sections))

	for i := from; i < to; i++ {
		l := m.lines[i]

		var text string
		if l.row < 0 {
			marker := "▸ "
			if m.sections[l.section].Expanded {
				marker = "▾ "
			}
			text = RenderSectionTitle(m.sections[l.section], strings.Repeat(" ", gutter)+helpStyle.Render(marker))
		} else {
			lines, ok := rowLines[l.section]
			if !ok {
				lines = RenderRows(m.sections[l.section].Rows, m.width-gutter-1)
				rowLines[l.section] = lines
			}
			// The same two columns the headings reserve, so rows sit under
			// their title rather than beside it
			text = strings.Repeat(" ", gutter) + lines[l.row]
		}

		if i == m.cursor {
			text = withCursor(text)
		}
		rendered = append(rendered, text)
	}
	return rendered
}

// withCursor draws the cursor in the gutter, which keeps every other column
// where it was.
func withCursor(text string) string {
	mark := cursorStyle.Render("›") + strings.Repeat(" ", gutter-1)
	if strings.HasPrefix(text, strings.Repeat(" ", gutter)) {
		return mark + text[gutter:]
	}
	return mark + text
}

// scrollTo keeps the cursor inside a window of the given height.
func (m *tuiModel) scrollTo(height int) {
	if len(m.lines) <= height {
		m.offset = 0
		return
	}
	m.offset = min(m.offset, m.cursor)
	if m.cursor >= m.offset+height {
		m.offset = m.cursor - height + 1
	}
	m.offset = min(m.offset, len(m.lines)-height)
}

// bodyHeight is what is left for rows once the header and footer are drawn.
func (m *tuiModel) bodyHeight() int {
	return m.bodyHeightBelow(RenderHeader(m.header))
}

// bodyHeightBelow takes the header already rendered, so a frame does not build
// the whole bordered box several times just to count its lines.
func (m *tuiModel) bodyHeightBelow(header string) int {
	const chrome = 4 // two blank lines and the two footer lines
	return max(m.height-lipgloss.Height(header)-chrome, 1)
}

// footer takes the body height its caller already computed, rather than
// rebuilding the header a second time to derive it.
func (m *tuiModel) footer(height int) string {
	help := "↑↓ move · → unfold · ← fold · a all · q quit"
	if action := m.currentAction(); action != nil {
		help = "⏎ " + action.Label + " · " + help
	}
	if len(m.lines) > height {
		help += " · pgup/pgdn scroll"
	}

	// What the last action did outranks the line the cursor happens to be on
	second := m.notice
	tone := m.noticeTone
	if second == "" {
		tone = ToneNeutral
		// Whatever the layout had to cut is worth having in full somewhere
		if m.cursor < len(m.lines) {
			if l := m.lines[m.cursor]; l.row >= 0 {
				row := m.sections[l.section].Rows[l.row]
				second = strings.TrimSpace(row.Key + " " + row.Value)
			}
		}
	}

	style := fullStyle
	if tone != ToneNeutral {
		style = toneStyles[tone]
	}
	return helpStyle.Render(truncate(help, m.width-1)) + "\n" + style.Render(truncate(second, m.width-1))
}

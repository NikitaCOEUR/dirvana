package status

import (
	"bytes"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestModel builds the interactive view over a config with two aliases and
// enough state for every section to appear.
func newTestModel(t *testing.T) *tuiModel {
	t.Helper()

	data := emptyData()
	data.HasAnyConfig = true
	data.HookInstalled = true
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Loaded: true, Authorized: true}}
	data.Aliases = map[string]AliasInfo{"k": {Command: "kubectl"}, "tf": {Command: "terraform"}}
	data.EnvStatic = map[string]string{"PROJECT": "dirvana"}

	m := &tuiModel{
		header:   BuildHeader(data),
		sections: BuildSections(data),
		width:    80,
		height:   40,
	}
	m.rebuild()
	return m
}

// press sends a key by name, the way Update does once bubbletea has decoded it.
func press(m *tuiModel, key string) {
	m.handleKey(key)
}

func TestTUI_StartsOnConfigAndAliases(t *testing.T) {
	m := newTestModel(t)

	view := m.render()
	assert.Contains(t, view, "kubectl", "aliases start unfolded")
	assert.NotContains(t, view, "PROJECT", "environment starts folded")

	// A folded section still reports what it holds
	assert.Contains(t, view, "1 variable")
}

func TestTUI_FoldAndUnfold(t *testing.T) {
	m := newTestModel(t)

	// Cursor sits on the Config heading; fold it
	press(m, "left")
	assert.NotContains(t, m.render(), ".dirvana.yml")

	press(m, "right")
	assert.Contains(t, m.render(), ".dirvana.yml")

	press(m, "enter")
	assert.NotContains(t, m.render(), ".dirvana.yml")

	press(m, " ")
	assert.Contains(t, m.render(), ".dirvana.yml")
}

func TestTUI_ToggleAll(t *testing.T) {
	m := newTestModel(t)

	press(m, "a")
	assert.Contains(t, m.render(), "PROJECT", "everything unfolds while something is folded")

	press(m, "a")
	view := m.render()
	assert.NotContains(t, view, "PROJECT")
	assert.NotContains(t, view, "kubectl", "a second press folds everything back")
}

func TestTUI_FoldingFromARowActsOnItsSection(t *testing.T) {
	m := newTestModel(t)

	press(m, "down") // onto the config file row
	require.GreaterOrEqual(t, m.lines[m.cursor].row, 0, "expected to be on a row, not a heading")

	press(m, "left")
	assert.NotContains(t, m.render(), ".dirvana.yml")
	// The cursor cannot stay on a line that no longer exists
	assert.Less(t, m.cursor, len(m.lines))
	assert.Equal(t, -1, m.lines[m.cursor].row)
}

func TestTUI_CursorStaysInRange(t *testing.T) {
	m := newTestModel(t)

	for range 100 {
		press(m, "up")
	}
	assert.Equal(t, 0, m.cursor)

	for range 100 {
		press(m, "down")
	}
	assert.Equal(t, len(m.lines)-1, m.cursor)

	press(m, "g")
	assert.Equal(t, 0, m.cursor)
	press(m, "G")
	assert.Equal(t, len(m.lines)-1, m.cursor)
}

func TestTUI_ScrollsToKeepCursorVisible(t *testing.T) {
	m := newTestModel(t)
	m.height = 12
	press(m, "a") // unfold everything, so there is more than fits

	for range len(m.lines) {
		press(m, "down")
		body := strings.Split(m.render(), "\n")
		assert.NotEmpty(t, body)
		assert.GreaterOrEqual(t, m.cursor, m.offset, "cursor scrolled off the top")
		assert.Less(t, m.cursor, m.offset+m.bodyHeight(), "cursor scrolled off the bottom")
	}
}

func TestTUI_ShowsCursorMark(t *testing.T) {
	m := newTestModel(t)

	assert.Contains(t, m.render(), "›")
}

func TestTUI_FooterShowsTheSelectedRowInFull(t *testing.T) {
	m := newTestModel(t)
	m.width = 30

	press(m, "down")
	press(m, "down")
	press(m, "down") // into the aliases

	line := m.lines[m.cursor]
	require.GreaterOrEqual(t, line.row, 0)
	row := m.sections[line.section].Rows[line.row]

	assert.Contains(t, m.render(), row.Value,
		"a value the layout had to cut must still be readable somewhere")
}

// newActionModel puts the view in front of a broken install, where the Setup
// section offers a fix.
func newActionModel(t *testing.T, run func() (string, error)) *tuiModel {
	t.Helper()

	data := emptyData()
	data.HookInstalled = false

	m := &tuiModel{data: data, header: BuildHeader(data), sections: BuildSections(data), width: 80, height: 40}
	m.sections[0].Rows[1].Action = &Action{Label: "install the hook", Run: run}
	m.rebuild()
	return m
}

func TestTUI_RunsTheFixFromTheRowReportingIt(t *testing.T) {
	ran := false
	m := newActionModel(t, func() (string, error) {
		ran = true
		return "✓ Hook installed", nil
	})

	press(m, "down")
	press(m, "down") // onto the hint row
	require.NotNil(t, m.currentAction(), "the hint row should offer the fix")

	// The key is advertised before it is pressed
	assert.Contains(t, m.render(), "⏎ install the hook")

	cmd := m.handleKey("enter")
	require.NotNil(t, cmd)
	m.Update(cmd())

	assert.True(t, ran)
	assert.Contains(t, m.render(), "✓ Hook installed")
}

func TestTUI_ReportsAFailedFix(t *testing.T) {
	m := newActionModel(t, func() (string, error) {
		return "", assert.AnError
	})

	press(m, "down")
	press(m, "down")
	m.Update(m.handleKey("enter")())

	assert.Contains(t, m.render(), assert.AnError.Error())
	assert.Equal(t, ToneError, m.noticeTone)
}

func TestTUI_ForgetsTheNoticeOnTheNextMove(t *testing.T) {
	m := newActionModel(t, func() (string, error) { return "done", nil })

	press(m, "down")
	press(m, "down")
	m.Update(m.handleKey("enter")())
	require.Contains(t, m.render(), "done")

	press(m, "up")
	assert.NotContains(t, m.render(), "done")
}

func TestTUI_KeepsFoldsAcrossARefresh(t *testing.T) {
	m := newActionModel(t, func() (string, error) { return "done", nil })
	m.data.Aliases = map[string]AliasInfo{"k": {Command: "kubectl"}}
	m.refreshSections()

	// Fold the aliases the user just folded, and prove a refresh respects it
	press(m, "a")
	press(m, "a")
	require.NotContains(t, m.render(), "kubectl")

	m.refreshSections()
	assert.NotContains(t, m.render(), "kubectl", "a refresh must not unfold what the user folded")
}

// TestTUI_CursorSurvivesASectionDisappearing covers the refresh that follows a
// fix: Setup drops out, every section below shifts up one index, and a cursor
// remembered by index would land on an unrelated line.
func TestTUI_CursorSurvivesASectionDisappearing(t *testing.T) {
	data := emptyData()
	data.HookInstalled = false
	data.HasAnyConfig = true
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Loaded: true, Authorized: true}}

	m := &tuiModel{data: data, header: BuildHeader(data), sections: BuildSections(data), width: 80, height: 40}
	m.rebuild()
	require.Equal(t, "Setup", m.sections[0].Title)

	// Move onto the config file row, two sections down
	for m.sections[m.lines[m.cursor].section].Title != "Config" || m.lines[m.cursor].row < 0 {
		press(m, "down")
	}

	// The fix worked: Setup has nothing left to report
	m.data.HookInstalled = true
	m.refreshSections()

	current := m.lines[m.cursor]
	assert.Equal(t, "Config", m.sections[current.section].Title,
		"the cursor must stay on the line it was on, not on whatever took its index")
	assert.Equal(t, 0, current.row)
}

func TestTUI_EnterStillFoldsWhereThereIsNoAction(t *testing.T) {
	m := newTestModel(t)

	press(m, "enter")
	assert.NotContains(t, m.render(), ".dirvana.yml")
}

func TestTUI_HeadingKeepsItsFoldMarkUnderTheCursor(t *testing.T) {
	m := newTestModel(t)

	// The cursor used to overwrite the marker, so the selected section no
	// longer showed whether it was folded
	view := m.render()
	assert.Contains(t, view, "›")
	assert.Contains(t, view, "▾")
}

func TestTUI_Quits(t *testing.T) {
	m := newTestModel(t)

	for _, key := range []string{"q", "esc", "ctrl+c"} {
		cmd := m.handleKey(key)
		require.NotNil(t, cmd, "%s should quit", key)
		assert.Equal(t, tea.Quit(), cmd(), "%s should quit", key)
	}
}

func TestTUI_ResizeIsHonoured(t *testing.T) {
	m := newTestModel(t)

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 50, m.height)
}

func TestTUI_IgnoresMessagesItDoesNotHandle(t *testing.T) {
	m := newTestModel(t)
	before := m.render()

	model, cmd := m.Update("something else entirely")

	assert.Same(t, m, model)
	assert.Nil(t, cmd)
	assert.Equal(t, before, m.render())
}

func TestTUI_UnknownKeysDoNothing(t *testing.T) {
	m := newTestModel(t)
	before := m.cursor

	assert.Nil(t, m.handleKey("ctrl+alt+z"))
	assert.Equal(t, before, m.cursor)
}

func TestTUI_PagingMovesByAScreen(t *testing.T) {
	m := newTestModel(t)
	m.height = 12
	press(m, "a") // unfold everything

	press(m, "pgdown")
	assert.Positive(t, m.cursor)

	press(m, "pgup")
	assert.Equal(t, 0, m.cursor)
}

// TestTUI_NothingToShow covers the degenerate model: no section at all, so
// every cursor lookup has to cope with an empty line list.
func TestTUI_NothingToShow(t *testing.T) {
	m := &tuiModel{data: &Data{}, sections: nil, width: 80, height: 24}
	m.rebuild()

	assert.Empty(t, m.lines)
	assert.Nil(t, m.currentAction())
	assert.Equal(t, -1, m.currentSection())
	assert.Equal(t, cursorTarget{row: -1}, m.cursorTarget())

	// None of these may panic on an empty view
	press(m, "down")
	press(m, "left")
	press(m, "right")
	press(m, "a")
	assert.NotEmpty(t, m.render())
}

func TestTUI_CursorOnALineWithoutAGutter(t *testing.T) {
	// Whatever the line looks like, the cursor has to be visible on it
	assert.Contains(t, withCursor("no gutter here"), "›")
	assert.Contains(t, withCursor("  gutter here"), "›")
}

func TestTUI_ViewRunsOnTheAlternateScreen(t *testing.T) {
	m := newTestModel(t)

	view := m.View()

	assert.True(t, view.AltScreen, "the terminal must be left as it was found")
	assert.Nil(t, m.Init())
}

// TestRunInteractive drives the real bubbletea program end to end, quitting on
// the first keystroke.
func TestRunInteractive(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Loaded: true, Authorized: true}}

	var out bytes.Buffer
	require.NoError(t, RunInteractive(data, strings.NewReader("q"), &out))
	assert.NotEmpty(t, out.String(), "the view must have been drawn")
}

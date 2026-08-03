package status

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emptyData returns data with every map and slice initialised, the shape
// CollectAll always hands over.
func emptyData() *Data {
	return &Data{
		CurrentDir:          "/test/dir",
		Version:             "1.0.0",
		Shell:               "bash",
		HookInstalled:       true,
		CachePath:           "/test/cache.json",
		AuthPath:            "/test/auth.json",
		Authorized:          true,
		Aliases:             make(map[string]AliasInfo),
		Functions:           make([]string, 0),
		EnvStatic:           make(map[string]string),
		EnvShell:            make(map[string]EnvShellInfo),
		Flags:               make([]string, 0),
		LocalConfigs:        make([]FileInfo, 0),
		CompletionScripts:   make([]CompletionScriptInfo, 0),
		CompletionOverrides: make(map[string]string),
	}
}

func TestRender_NoConfig(t *testing.T) {
	data := emptyData()

	output := Render(data)

	assert.Contains(t, output, "dirvana 1.0.0")
	assert.Contains(t, output, "/test/dir")
	assert.Contains(t, output, "bash, hook installed")
	assert.Contains(t, output, "no config here")

	// Nothing to say about aliases or configs here, so those sections are not
	// printed as empty shells
	assert.NotContains(t, output, "Aliases")
	assert.NotContains(t, output, "Config\n")
}

func TestRender_UnauthorizedConfig(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.Authorized = false
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml"}}

	output := Render(data)

	assert.Contains(t, output, "not authorized")
	assert.Contains(t, output, "/test/dir/.dirvana.yml")
	// The way out has to be on screen, not in the docs
	assert.Contains(t, output, "dirvana allow /test/dir")
}

func TestRender_OutdatedHook(t *testing.T) {
	data := emptyData()
	data.Shell = "fish"
	data.HookOutdated = true
	data.RCFile = "/home/user/.config/fish/config.fish"

	output := Render(data)

	assert.Contains(t, output, "fish, hook outdated")
	assert.Contains(t, output, "dirvana setup fish")
}

func TestRender_MissingHook(t *testing.T) {
	data := emptyData()
	data.Shell = "zsh"
	data.HookInstalled = false

	output := Render(data)

	assert.Contains(t, output, "zsh, hook not installed")
	// Nothing else in the output matters if the hook never runs
	assert.Contains(t, output, "dirvana setup zsh")
}

func TestRender_UndetectedShell(t *testing.T) {
	data := emptyData()
	data.Shell = ""

	// Reporting bash when nothing was detected is what made status claim a
	// missing hook under fish
	assert.Contains(t, Render(data), "shell not detected")
}

func TestRender_FullConfig(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.Shell = "zsh"
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Loaded: true, Authorized: true}}
	data.GlobalConfig = &GlobalInfo{Path: "/home/user/config.yml", Exists: true, Loaded: true}
	data.Aliases = map[string]AliasInfo{
		"gs": {Command: "git status"},
		"k":  {Command: "wrapper kubectl"},
	}
	data.CompletionOverrides = map[string]string{"k": "kubectl"}
	data.Functions = []string{"mkcd", "greet"}
	data.EnvStatic = map[string]string{"PROJECT": "dirvana"}
	data.EnvShell = map[string]EnvShellInfo{"BRANCH": {Command: "git branch --show-current"}}
	data.Flags = []string{"local_only"}
	data.CacheTotalEntries = 5
	data.CacheFileSize = 4096
	data.CacheValid = true
	data.CacheUpdated = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	output := Render(data)

	assert.Contains(t, output, "git status")
	// The completion an alias resolves to belongs next to the alias, not in a
	// separate section further down
	assert.Regexp(t, `k\s+wrapper kubectl\s+⇥ kubectl`, output)

	assert.Contains(t, output, "greet()")
	assert.Contains(t, output, "PROJECT")
	assert.Contains(t, output, "$(git branch --show-current)")
	assert.Contains(t, output, "not approved")
	assert.Contains(t, output, "local_only")
	assert.Contains(t, output, "4.0 KB")
	assert.Contains(t, output, "2024-01-01 12:00")
}

func TestRender_SortsEverything(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.Aliases = map[string]AliasInfo{
		"zeta": {Command: "z"}, "alpha": {Command: "a"}, "mid": {Command: "m"},
	}
	data.Functions = []string{"zfunc", "afunc"}
	data.EnvStatic = map[string]string{"ZVAR": "z", "AVAR": "a"}

	// Map iteration order is random: rendering twice used to give two
	// different listings, which makes the output impossible to diff
	first := Render(data)
	for range 10 {
		assert.Equal(t, first, Render(data))
	}

	assert.Less(t, strings.Index(first, "alpha"), strings.Index(first, "mid"))
	assert.Less(t, strings.Index(first, "mid"), strings.Index(first, "zeta"))
	assert.Less(t, strings.Index(first, "afunc"), strings.Index(first, "zfunc"))
	assert.Less(t, strings.Index(first, "AVAR"), strings.Index(first, "ZVAR"))
}

func TestRender_PathsAreListedOnce(t *testing.T) {
	data := emptyData()

	output := Render(data)

	// The cache path used to be printed by two different sections
	assert.Equal(t, 1, strings.Count(output, "/test/cache.json"))
}

func TestRenderRows_AlignsColumns(t *testing.T) {
	rows := []Row{
		{Key: "k", Value: "kubectl", Note: "⇥ kubectl"},
		{Key: "terraform", Value: "task terraform --", Note: "⇥ terraform"},
		{Key: "gs", Value: "git status"},
	}

	lines := RenderRows(rows, 0)
	require.Len(t, lines, 3)

	// Values start at the same column whatever the key length
	assert.Equal(t, strings.Index(lines[0], "kubectl"), strings.Index(lines[1], "task"))
	// And so do the notes, past the longest value
	assert.Equal(t, strings.Index(lines[0], "⇥"), strings.Index(lines[1], "⇥"))
}

func TestRenderRows_DetailRowsAreIndented(t *testing.T) {
	rows := []Row{
		{Key: "k", Value: "kubectl"},
		{Value: "when var:KUBECONFIG", Detail: true},
	}

	lines := RenderRows(rows, 0)

	assert.Greater(t, indentOf(lines[1]), indentOf(lines[0]),
		"a condition belongs under the alias it qualifies")
}

func TestRenderRows_ShortensToWidth(t *testing.T) {
	rows := []Row{{Key: "k", Value: strings.Repeat("x", 200), Note: "⇥ kubectl"}}

	line := RenderRows(rows, 40)[0]

	assert.LessOrEqual(t, len([]rune(stripANSI(line))), 40)
	// The note is what says where a completion comes from; it must survive the
	// value being cut
	assert.Contains(t, line, "⇥ kubectl")
	assert.Contains(t, line, "…")
}

// TestRenderRows_NeverExceedsWidth is the property the whole layout rests on:
// the TUI counts one screen line per row, so a row that overflows wraps and
// pushes the footer out of view. Honouring only the note column let a long key
// blow past the right edge.
func TestRenderRows_NeverExceedsWidth(t *testing.T) {
	rows := []Row{
		{Key: strings.Repeat("K", 44), Value: strings.Repeat("v", 41), Note: "not approved"},
		{Key: "short", Value: strings.Repeat("v", 200), Note: "⇥ kubectl"},
		{Key: strings.Repeat("K", 60), Value: "x"},
		{Value: strings.Repeat("d", 120), Detail: true},
		{Key: strings.Repeat("K", 30)},
	}

	for _, width := range []int{20, 37, 50, 80, 120} {
		for i, line := range RenderRows(rows, width) {
			assert.LessOrEqual(t, lipgloss.Width(stripANSI(line)), width,
				"row %d overflows at width %d: %q", i, width, stripANSI(line))
		}
	}
}

func TestRenderRows_KeepsTheNoteWhenSpaceIsTight(t *testing.T) {
	rows := []Row{{Key: "KUBECONFIG", Value: strings.Repeat("v", 80), Note: "not approved"}}

	line := stripANSI(RenderRows(rows, 40)[0])

	// The note is the actionable half of the row; the value is what gives way
	assert.Contains(t, line, "not approved")
	assert.LessOrEqual(t, lipgloss.Width(line), 40)
}

func TestTruncate_NeverGrowsPastWidth(t *testing.T) {
	for _, s := range []string{"abc", "élan vital", "日本語のテキスト", strings.Repeat("x", 50)} {
		for width := range 12 {
			assert.LessOrEqual(t, lipgloss.Width(truncate(s, width)), width,
				"truncate(%q, %d)", s, width)
		}
	}
}

func TestRenderRows_NoNotesNoPadding(t *testing.T) {
	rows := []Row{{Key: "a", Value: "one"}, {Key: "b", Value: "two"}}

	for _, line := range RenderRows(rows, 0) {
		assert.Equal(t, line, strings.TrimRight(line, " "), "no trailing padding without notes")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"truncate me please", 10, "truncate…"},
		{"abc", 1, "…"},
		{"élan vital", 5, "élan…"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, truncate(tt.input, tt.width), "truncate(%q, %d)", tt.input, tt.width)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatBytes(tt.input), "formatBytes(%d)", tt.input)
	}
}

// indentOf counts the leading spaces of a rendered line, escapes aside.
func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(stripANSI(line), " "))
}

// stripANSI removes escape sequences so tests can measure visible text.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

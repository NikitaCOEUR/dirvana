package status

import (
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/setup"
	shellpkg "github.com/NikitaCOEUR/dirvana/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findSection returns the section with that title, failing the test when the
// layout no longer produces it.
func findSection(t *testing.T, sections []Section, title string) Section {
	t.Helper()
	for _, s := range sections {
		if s.Title == title {
			return s
		}
	}
	t.Fatalf("no %q section in %v", title, sectionTitles(sections))
	return Section{}
}

func sectionTitles(sections []Section) []string {
	titles := make([]string, 0, len(sections))
	for _, s := range sections {
		titles = append(titles, s.Title)
	}
	return titles
}

func TestBuildSections_DropsEmptyOnes(t *testing.T) {
	data := emptyData()

	// Cache and paths always have something to report; the rest does not, and
	// a working install has nothing to say about setup
	assert.Equal(t, []string{"Cache", "Paths"}, sectionTitles(BuildSections(data)))
}

func TestBuildSections_SetupOnlyWhenSomethingIsWrong(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Data)
		summary string
		hint    string
	}{
		{"missing hook", func(d *Data) { d.HookInstalled = false }, "hook not installed", "dirvana setup bash"},
		{"outdated hook", func(d *Data) { d.HookOutdated = true }, "hook outdated", "dirvana setup bash"},
		{"no shell", func(d *Data) { d.Shell = "" }, "shell not detected", "dirvana setup <bash|zsh|fish>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := emptyData()
			tt.mutate(data)

			section := findSection(t, BuildSections(data), "Setup")
			assert.Equal(t, tt.summary, section.Summary)
			assert.True(t, section.Expanded, "a broken install must not be hidden behind a fold")
			require.Len(t, section.Rows, 2)
			assert.Contains(t, section.Rows[1].Value, tt.hint)
		})
	}
}

func TestBuildSections_OpensOnConfigAndAliases(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Loaded: true, Authorized: true}}
	data.Aliases = map[string]AliasInfo{"k": {Command: "kubectl"}}

	for _, section := range BuildSections(data) {
		expanded := section.Title == "Config" || section.Title == "Aliases"
		assert.Equal(t, expanded, section.Expanded,
			"%s should start %s", section.Title, map[bool]string{true: "unfolded", false: "folded"}[expanded])
	}
}

func TestBuildSections_FoldedSectionsStillSaySomething(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.CacheTotalEntries = 140
	data.CacheFileSize = 91 * 1024
	data.CompletionDetection = &CompletionDetectionInfo{
		Commands: map[string]string{"kubectl": "Cobra", "helm": "Cobra", "task": "Script"},
	}

	sections := BuildSections(data)

	cache := findSection(t, sections, "Cache")
	assert.Equal(t, "140 entries, 91.0 KB, this directory stale", cache.Summary)
	assert.Equal(t, "3 commands detected", findSection(t, sections, "Completion").Summary)
}

func TestBuildSections_AliasCarriesItsCompletion(t *testing.T) {
	data := emptyData()
	data.Aliases = map[string]AliasInfo{
		"k":  {Command: "/opt/bin/wrapper.sh kubectl"},
		"gs": {Command: "git status"},
	}
	data.CompletionOverrides = map[string]string{"k": "kubectl"}

	rows := findSection(t, BuildSections(data), "Aliases").Rows
	require.Len(t, rows, 2)

	// Sorted, and the completion target sits on the alias it applies to
	assert.Equal(t, "gs", rows[0].Key)
	assert.Empty(t, rows[0].Note)
	assert.Equal(t, "k", rows[1].Key)
	assert.Equal(t, "⇥ kubectl", rows[1].Note)
}

func TestBuildSections_RedundantCompletionIsNotShown(t *testing.T) {
	data := emptyData()
	data.Aliases = map[string]AliasInfo{
		"g":  {Command: "git"},
		"kc": {Command: "/opt/bin/kubectl --context prod"},
	}
	data.CompletionOverrides = map[string]string{"g": "git", "kc": "kubectl"}

	rows := findSection(t, BuildSections(data), "Aliases").Rows
	require.Len(t, rows, 2)

	// Saying that `git` completes as git, or that a kubectl wrapper completes
	// as kubectl, is noise on every single line
	assert.Empty(t, rows[0].Note)
	assert.Empty(t, rows[1].Note)
}

func TestBuildSections_ConditionalAlias(t *testing.T) {
	data := emptyData()
	data.Aliases = map[string]AliasInfo{
		"k": {
			Command:     "kubectl",
			HasWhen:     true,
			WhenSummary: "var:KUBECONFIG",
			Else:        "echo no cluster",
		},
	}

	rows := findSection(t, BuildSections(data), "Aliases").Rows
	require.Len(t, rows, 3)

	assert.Equal(t, "k", rows[0].Key)
	assert.True(t, rows[1].Detail)
	assert.Equal(t, "when var:KUBECONFIG", rows[1].Value)
	assert.True(t, rows[2].Detail)
	assert.Equal(t, "else echo no cluster", rows[2].Value)
}

func TestBuildSections_ShellEnvApproval(t *testing.T) {
	data := emptyData()
	data.EnvShell = map[string]EnvShellInfo{
		"OK":  {Command: "date", Approved: true},
		"NOK": {Command: "whoami"},
	}

	rows := findSection(t, BuildSections(data), "Environment").Rows
	require.Len(t, rows, 2)

	assert.Equal(t, "NOK", rows[0].Key)
	assert.Equal(t, "$(whoami)", rows[0].Value)
	assert.Equal(t, "not approved", rows[0].Note)
	assert.Equal(t, ToneWarn, rows[0].Tone)
	assert.Equal(t, "approved", rows[1].Note)
}

func TestBuildSections_UnauthorizedConfigTellsWhatToRun(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.Authorized = false
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml"}}

	section := findSection(t, BuildSections(data), "Config")
	require.Len(t, section.Rows, 2)

	assert.Equal(t, "not authorized", section.Rows[0].Note)
	assert.Equal(t, ToneError, section.Rows[0].Tone)
	assert.Contains(t, section.Rows[1].Value, "dirvana allow /test/dir")
	// The hint is not a config file
	assert.Equal(t, "1 file", section.Summary)
}

func TestBuildSections_GlobalConfigComesFirst(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.GlobalConfig = &GlobalInfo{Path: "/home/user/config.yml", Exists: true, Loaded: true}
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Loaded: true, Authorized: true}}

	rows := findSection(t, BuildSections(data), "Config").Rows
	require.Len(t, rows, 2)

	assert.Equal(t, "global", rows[0].Note)
	assert.Contains(t, rows[1].Value, ".dirvana.yml")
}

func TestBuildSections_IgnoredGlobalConfig(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.GlobalConfig = &GlobalInfo{Path: "/home/user/config.yml", Exists: true, Loaded: false}

	rows := findSection(t, BuildSections(data), "Config").Rows
	require.Len(t, rows, 1)

	assert.Equal(t, "ignored", rows[0].Note)
	assert.Equal(t, ToneWarn, rows[0].Tone)
}

func TestBuildSections_CompletionGroupsBySource(t *testing.T) {
	data := emptyData()
	data.CompletionDetection = &CompletionDetectionInfo{
		Commands: map[string]string{
			"kubectl": "Cobra", "helm": "Cobra", "argocd": "Flag", "task": "Script",
		},
	}
	data.CompletionRegistry = &CompletionRegistryInfo{Path: "/reg.yml", Size: 1024, ToolsCount: 3}

	rows := findSection(t, BuildSections(data), "Completion").Rows
	require.GreaterOrEqual(t, len(rows), 3)

	assert.Equal(t, "Cobra", rows[0].Key)
	assert.Equal(t, "helm, kubectl", rows[0].Value, "commands listed in a stable order")
	assert.Equal(t, "2", rows[0].Note)
	assert.Equal(t, "Flag", rows[1].Key)
	assert.Equal(t, "Script", rows[2].Key)
	assert.Equal(t, "3 tools available", rows[3].Value)
}

func TestBuildHeader(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Data)
		expected string
	}{
		{"hook missing", func(d *Data) { d.Shell = "fish"; d.HookInstalled = false }, "fish, hook not installed"},
		{"hook installed", func(d *Data) { d.Shell = "fish" }, "fish, hook installed"},
		{"hook outdated", func(d *Data) { d.Shell = "zsh"; d.HookOutdated = true }, "zsh, hook outdated"},
		{"no shell", func(d *Data) { d.Shell = "" }, "shell not detected"},
		{"authorized", func(d *Data) { d.HasAnyConfig = true }, "authorized"},
		{"unauthorized", func(d *Data) { d.HasAnyConfig = true; d.Authorized = false }, "not authorized"},
		{"no config", func(_ *Data) {}, "no config here"},
		{"flags", func(d *Data) { d.Flags = []string{"local_only"} }, "local_only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := emptyData()
			tt.mutate(data)

			texts := make([]string, 0)
			for _, chip := range BuildHeader(data).Chips {
				texts = append(texts, chip.Text)
			}
			assert.Contains(t, texts, tt.expected)
		})
	}
}

func TestShorten(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Home is replaced wherever it appears, including inside a command, since
	// that is where the long paths actually are
	assert.Equal(t, "~/Github/fugu", shorten(home+"/Github/fugu"))
	assert.Equal(t, "~/bin/wrap.sh kubectl", shorten(home+"/bin/wrap.sh kubectl"))
	assert.Equal(t, "/etc/hosts", shorten("/etc/hosts"))
}

func TestCountLabel(t *testing.T) {
	assert.Equal(t, "1 file", countLabel(1, "file"))
	assert.Equal(t, "0 files", countLabel(0, "file"))
	assert.Equal(t, "2 files", countLabel(2, "file"))
	assert.Equal(t, "3 aliases", countLabel(3, "alias", "aliases"))
	assert.Equal(t, "1 alias", countLabel(1, "alias", "aliases"))
}

// TestInstallHookAction runs the action the Setup row offers, against a home
// directory of its own.
func TestInstallHookAction(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	action := installHookAction("fish", "install the hook")
	require.NotNil(t, action)
	assert.Equal(t, "install the hook", action.Label)

	message, err := action.Run()
	require.NoError(t, err)
	assert.NotEmpty(t, message)
	assert.NotContains(t, message, "\n", "the view has one line for it")

	// And it really installed something
	state, err := setup.InspectHook("fish")
	require.NoError(t, err)
	assert.True(t, state.Installed)
}

func TestInstallHookAction_ReportsFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := installHookAction("nushell", "install the hook").Run()
	assert.Error(t, err)
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "one", firstLine("one\ntwo\nthree"))
	assert.Equal(t, "alone", firstLine("alone"))
	assert.Empty(t, firstLine(""))
}

func TestBuildSections_BrokenHook(t *testing.T) {
	data := emptyData()
	data.HookInstalled = true
	data.HookBroken = true
	data.HookBinary = "/gone/dirvana"
	data.HookFile = "/home/user/.config/dirvana/hook-bash.sh"

	section := findSection(t, BuildSections(data), "Setup")

	assert.Equal(t, "hook broken", section.Summary)
	assert.Contains(t, section.Rows[0].Value, "/gone/dirvana")
	assert.Equal(t, ToneError, section.Rows[0].Tone)
	// The fix names the binary it would write in, since it is not the one there
	assert.Contains(t, section.Rows[1].Value, "calling ")
}

func TestBuildSections_FixRowStaysQuietWhenTheBinaryIsTheSameOne(t *testing.T) {
	data := emptyData()
	data.HookInstalled = true
	data.HookOutdated = true
	data.HookBinary = shellpkg.BinaryPath()

	section := findSection(t, BuildSections(data), "Setup")

	assert.NotContains(t, section.Rows[1].Value, "calling ")
}

func TestBuildSections_CacheLocalOnly(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.CacheValid = true
	data.CacheLocalOnly = true

	rows := findSection(t, BuildSections(data), "Cache").Rows

	assert.True(t, rows[len(rows)-1].Detail)
	assert.Contains(t, rows[len(rows)-1].Value, "local only")
}

func TestBuildSections_CompletionNotProbedYet(t *testing.T) {
	data := emptyData()
	data.CompletionRegistry = &CompletionRegistryInfo{Path: "/reg.yml", Size: 10}

	section := findSection(t, BuildSections(data), "Completion")

	assert.Equal(t, "nothing detected yet", section.Summary)
	assert.Contains(t, section.Rows[0].Value, "detection happens on first completion")
}

func TestBuildSections_DownloadedScripts(t *testing.T) {
	data := emptyData()
	data.CompletionDetection = &CompletionDetectionInfo{Commands: map[string]string{"helm": "Cobra"}}
	data.CompletionScripts = []CompletionScriptInfo{{Tool: "kubectl", Size: 2048}}

	rows := findSection(t, BuildSections(data), "Completion").Rows

	last := rows[len(rows)-1]
	assert.Equal(t, "Script", last.Key)
	assert.Equal(t, "kubectl", last.Value)
	assert.Equal(t, "2.0 KB", last.Note)
}

func TestBuildSections_NotLoadedConfig(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Authorized: true, Loaded: false}}

	rows := findSection(t, BuildSections(data), "Config").Rows

	assert.Equal(t, "not loaded", rows[0].Note)
	assert.Equal(t, ToneWarn, rows[0].Tone)
}

func TestBuildSections_LocalOnlyConfig(t *testing.T) {
	data := emptyData()
	data.HasAnyConfig = true
	data.LocalConfigs = []FileInfo{{Path: "/test/dir/.dirvana.yml", Authorized: true, Loaded: true, LocalOnly: true}}

	rows := findSection(t, BuildSections(data), "Config").Rows

	assert.Equal(t, "loaded, local only", rows[0].Note)
}

func TestFirstWord(t *testing.T) {
	assert.Equal(t, "kubectl", firstWord("kubectl get pods"))
	assert.Equal(t, "wrapper.sh", firstWord("/opt/bin/wrapper.sh kubectl"))
	assert.Empty(t, firstWord(""))
	assert.Empty(t, firstWord("   "))
}

func TestBuildHeader_BrokenHook(t *testing.T) {
	data := emptyData()
	data.HookInstalled = true
	data.HookBroken = true

	texts := make([]string, 0)
	for _, chip := range BuildHeader(data).Chips {
		texts = append(texts, chip.Text)
	}
	assert.Contains(t, texts, "bash, hook broken")
}

func TestShorten_WithoutAUsableHome(t *testing.T) {
	// A home of "/" would turn every absolute path into a bare "~"
	t.Setenv("HOME", "/")
	assert.Equal(t, "/etc/hosts", shorten("/etc/hosts"))

	t.Setenv("HOME", "")
	assert.Equal(t, "/etc/hosts", shorten("/etc/hosts"))
}

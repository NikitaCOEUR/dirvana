package status

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/setup"
	shellpkg "github.com/NikitaCOEUR/dirvana/internal/shell"
)

// Tone qualifies a piece of text so every renderer colours it the same way.
type Tone int

// Tones, from neutral to alarming.
const (
	ToneNeutral Tone = iota
	ToneMuted
	ToneOK
	ToneWarn
	ToneError
)

// Chip is a short piece of state shown on the header line.
type Chip struct {
	Text string
	Tone Tone
}

// Header is the always-visible summary: what dirvana is, where it is looking,
// and the handful of facts worth seeing before anything is unfolded.
type Header struct {
	Version string
	Dir     string
	Chips   []Chip
}

// Row is one line of a section: a key, its value, and an optional note pinned
// to the right of the line.
type Row struct {
	Key   string
	Value string
	Note  string
	Tone  Tone
	// Detail marks a continuation line - a condition, a hint - rendered
	// indented under the row it belongs to.
	Detail bool
	// Action, when set, is what the interactive view runs on this row. Rows
	// that only state a fact leave it nil.
	Action *Action
}

// Action is a fix the interactive view can apply from the row that reports the
// problem, instead of sending the user off to type the command themselves.
type Action struct {
	// Label says what running it does, in the imperative.
	Label string
	// Run performs it and returns what to tell the user.
	Run func() (string, error)
}

// Section is a foldable group of rows. Summary stands in for the rows while
// the section is folded, so a collapsed view still says something useful.
type Section struct {
	Title    string
	Icon     string
	Summary  string
	Rows     []Row
	Expanded bool
}

// BuildHeader turns the collected data into the header line.
func BuildHeader(data *Data) Header {
	h := Header{
		Version: data.Version,
		Dir:     shorten(data.CurrentDir),
	}

	switch {
	case data.Shell == "":
		h.Chips = append(h.Chips, Chip{"shell not detected", ToneWarn})
	case !data.HookInstalled:
		h.Chips = append(h.Chips, Chip{data.Shell + ", hook not installed", ToneError})
	case data.HookBroken:
		h.Chips = append(h.Chips, Chip{data.Shell + ", hook broken", ToneError})
	case data.HookOutdated:
		h.Chips = append(h.Chips, Chip{data.Shell + ", hook outdated", ToneWarn})
	default:
		h.Chips = append(h.Chips, Chip{data.Shell + ", hook installed", ToneOK})
	}

	switch {
	case !data.HasAnyConfig:
		h.Chips = append(h.Chips, Chip{"no config here", ToneMuted})
	case data.Authorized:
		h.Chips = append(h.Chips, Chip{"authorized", ToneOK})
	default:
		h.Chips = append(h.Chips, Chip{"not authorized", ToneError})
	}

	for _, flag := range data.Flags {
		h.Chips = append(h.Chips, Chip{flag, ToneMuted})
	}

	return h
}

// BuildSections lays the collected data out as foldable sections, in the order
// they are shown. Sections with nothing to say are left out entirely.
func BuildSections(data *Data) []Section {
	sections := []Section{
		setupSection(data),
		configSection(data),
		aliasSection(data),
		functionSection(data),
		envSection(data),
		cacheSection(data),
		completionSection(data),
		pathSection(data),
	}

	kept := make([]Section, 0, len(sections))
	for _, s := range sections {
		if len(s.Rows) > 0 {
			kept = append(kept, s)
		}
	}
	return kept
}

// setupSection only exists when something stands between dirvana and a working
// shell. When the hook is in place it has nothing to say and is dropped.
func setupSection(data *Data) Section {
	s := Section{Title: "Setup", Icon: "⚠️", Expanded: true}

	switch {
	case data.Shell == "":
		s.Summary = "shell not detected"
		s.Rows = append(s.Rows,
			Row{Value: "dirvana could not tell which shell is running", Tone: ToneWarn},
			Row{Value: "run 'dirvana setup <bash|zsh|fish>' to install the hook", Tone: ToneMuted, Detail: true},
		)
		return s
	case !data.HookInstalled:
		s.Summary = "hook not installed"
		s.Rows = append(s.Rows,
			Row{Value: "aliases and completions are not loaded on cd", Tone: ToneError},
			hookFixRow(data, "install the hook"),
		)
	case data.HookBroken:
		s.Summary = "hook broken"
		s.Rows = append(s.Rows,
			Row{Value: "the hook calls " + shorten(data.HookBinary) + ", which is gone", Tone: ToneError},
			hookFixRow(data, "repair the hook"),
		)
	case data.HookOutdated:
		s.Summary = "hook outdated"
		s.Rows = append(s.Rows,
			Row{Value: "the installed hook is from another version of dirvana", Tone: ToneWarn},
			hookFixRow(data, "refresh the hook"),
		)
	default:
		return s
	}

	return s
}

// hookFixRow is the line offering to run the installer, naming the dirvana it
// would write into the hook whenever that is not the one already there - a
// binary run from a download would otherwise be baked in silently.
func hookFixRow(data *Data, label string) Row {
	row := Row{
		Value:  fmt.Sprintf("run 'dirvana setup %s' to write the hook to %s", data.Shell, shorten(data.HookFile)),
		Tone:   ToneMuted,
		Detail: true,
		Action: installHookAction(data.Shell, label),
	}

	if binary := shellpkg.BinaryPath(); binary != data.HookBinary {
		row.Value += ", calling " + shorten(binary)
	}
	return row
}

// installHookAction wires the row to the same installer `dirvana setup` runs,
// so the fix is one keystroke away from the line reporting the problem.
func installHookAction(shellName, label string) *Action {
	return &Action{
		Label: label,
		Run: func() (string, error) {
			result, err := setup.InstallHook(shellName)
			if err != nil {
				return "", err
			}
			// The installer's message is written for a scrolling terminal;
			// the view has one line for it
			return firstLine(result.Message), nil
		},
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func configSection(data *Data) Section {
	s := Section{Title: "Config", Icon: "📝", Expanded: true}

	if data.GlobalConfig != nil && data.GlobalConfig.Exists {
		row := Row{Value: shorten(data.GlobalConfig.Path), Note: "global", Tone: ToneMuted}
		if !data.GlobalConfig.Loaded {
			row.Note = "ignored"
			row.Tone = ToneWarn
		}
		s.Rows = append(s.Rows, row)
	}

	for _, cfg := range data.LocalConfigs {
		row := Row{Value: shorten(cfg.Path), Note: "loaded", Tone: ToneOK}
		switch {
		case !cfg.Authorized:
			row.Note, row.Tone = "not authorized", ToneError
		case !cfg.Loaded:
			row.Note, row.Tone = "not loaded", ToneWarn
		case cfg.LocalOnly:
			row.Note = "loaded, local only"
		}
		s.Rows = append(s.Rows, row)
	}

	if len(s.Rows) == 0 {
		return s
	}

	if data.HasAnyConfig && !data.Authorized {
		s.Rows = append(s.Rows, Row{
			Value:  fmt.Sprintf("run 'dirvana allow %s' to load them", shorten(data.CurrentDir)),
			Tone:   ToneWarn,
			Detail: true,
		})
	}

	s.Summary = countLabel(len(s.Rows), "file")
	if data.HasAnyConfig && !data.Authorized {
		// The hint row is not a file
		s.Summary = countLabel(len(s.Rows)-1, "file")
	}
	return s
}

func aliasSection(data *Data) Section {
	s := Section{Title: "Aliases", Icon: "🔗", Expanded: true}

	for _, name := range slices.Sorted(maps.Keys(data.Aliases)) {
		info := data.Aliases[name]
		row := Row{Key: name, Value: shorten(info.Command)}
		if target, ok := data.CompletionOverrides[name]; ok && target != firstWord(info.Command) {
			// Worth saying only when the completion comes from somewhere else
			// than the command being run: `tf` runs task but completes as
			// terraform, whereas `g` completing as git says nothing.
			row.Note = "⇥ " + target
			row.Tone = ToneMuted
		}
		s.Rows = append(s.Rows, row)

		if info.HasWhen {
			s.Rows = append(s.Rows, Row{Value: "when " + info.WhenSummary, Tone: ToneMuted, Detail: true})
			if info.Else != "" {
				s.Rows = append(s.Rows, Row{Value: "else " + shorten(info.Else), Tone: ToneMuted, Detail: true})
			}
		}
	}

	s.Summary = countLabel(len(data.Aliases), "alias", "aliases")
	return s
}

func functionSection(data *Data) Section {
	s := Section{Title: "Functions", Icon: "🧩"}

	for _, name := range slices.Sorted(slices.Values(data.Functions)) {
		s.Rows = append(s.Rows, Row{Key: name + "()"})
	}

	s.Summary = countLabel(len(s.Rows), "function")
	return s
}

func envSection(data *Data) Section {
	s := Section{Title: "Environment", Icon: "🌍"}

	for _, name := range slices.Sorted(maps.Keys(data.EnvStatic)) {
		s.Rows = append(s.Rows, Row{Key: name, Value: shorten(data.EnvStatic[name])})
	}

	for _, name := range slices.Sorted(maps.Keys(data.EnvShell)) {
		v := data.EnvShell[name]
		row := Row{Key: name, Value: "$(" + v.Command + ")", Note: "approved", Tone: ToneOK}
		if !v.Approved {
			row.Note, row.Tone = "not approved", ToneWarn
		}
		s.Rows = append(s.Rows, row)
	}

	s.Summary = countLabel(len(s.Rows), "variable")
	return s
}

func cacheSection(data *Data) Section {
	s := Section{Title: "Cache", Icon: "💾"}

	s.Rows = append(s.Rows,
		Row{Key: "Entries", Value: fmt.Sprintf("%d", data.CacheTotalEntries)},
		Row{Key: "Size", Value: formatBytes(data.CacheFileSize)},
	)

	summary := fmt.Sprintf("%s, %s", countLabel(data.CacheTotalEntries, "entry", "entries"), formatBytes(data.CacheFileSize))

	if data.HasAnyConfig {
		if data.CacheValid {
			s.Rows = append(s.Rows, Row{
				Key:   "This directory",
				Value: "up to date, generated " + data.CacheUpdated.Format("2006-01-02 15:04"),
				Tone:  ToneOK,
			})
			if data.CacheLocalOnly {
				s.Rows = append(s.Rows, Row{Value: "local only: parent configs are not merged", Tone: ToneMuted, Detail: true})
			}
			summary += ", this directory up to date"
		} else {
			s.Rows = append(s.Rows, Row{
				Key:   "This directory",
				Value: "stale, rebuilt on the next cd",
				Tone:  ToneWarn,
			})
			summary += ", this directory stale"
		}
	}

	s.Summary = summary
	return s
}

func completionSection(data *Data) Section {
	s := Section{Title: "Completion", Icon: "🔄"}

	if data.CompletionDetection == nil || len(data.CompletionDetection.Commands) == 0 {
		if data.CompletionRegistry == nil {
			return s
		}
		s.Summary = "nothing detected yet"
		s.Rows = append(s.Rows, Row{Value: "no command probed yet; detection happens on first completion", Tone: ToneMuted})
	} else {
		// Group by how the completion was found, so an unexpected source is
		// visible at a glance
		bySource := make(map[string][]string)
		for cmd, source := range data.CompletionDetection.Commands {
			bySource[source] = append(bySource[source], cmd)
		}
		for _, source := range []string{"Cobra", "Flag", "Env", "Script"} {
			cmds, ok := bySource[source]
			if !ok {
				continue
			}
			slices.Sort(cmds)
			s.Rows = append(s.Rows, Row{
				Key:   source,
				Value: strings.Join(cmds, ", "),
				Note:  fmt.Sprintf("%d", len(cmds)),
				Tone:  ToneMuted,
			})
		}
		s.Summary = countLabel(len(data.CompletionDetection.Commands), "command") + " detected"
	}

	if data.CompletionRegistry != nil && data.CompletionRegistry.ToolsCount > 0 {
		s.Rows = append(s.Rows, Row{
			Key:   "Registry",
			Value: countLabel(data.CompletionRegistry.ToolsCount, "tool") + " available",
		})
	}

	for _, script := range data.CompletionScripts {
		s.Rows = append(s.Rows, Row{
			Key:   "Script",
			Value: script.Tool,
			Note:  formatBytes(script.Size),
			Tone:  ToneMuted,
		})
	}

	return s
}

// pathSection collects everything that is only useful when something needs to
// be inspected or deleted by hand.
func pathSection(data *Data) Section {
	s := Section{Title: "Paths", Icon: "📁", Summary: "cache, auth and completion files"}

	s.Rows = append(s.Rows,
		Row{Key: "Cache", Value: shorten(data.CachePath), Tone: ToneMuted},
		Row{Key: "Auth", Value: shorten(data.AuthPath), Tone: ToneMuted},
	)

	// The hook code and the config pulling it in are two different files for
	// every strategy but the inline one, and it is the former the user needs
	// when reading or removing the hook by hand
	if data.HookFile != "" && data.HookFile != data.RCFile {
		s.Rows = append(s.Rows, Row{Key: "Hook", Value: shorten(data.HookFile), Tone: ToneMuted})
	}
	if data.RCFile != "" {
		s.Rows = append(s.Rows, Row{Key: "Shell config", Value: shorten(data.RCFile), Tone: ToneMuted})
	}
	if data.CompletionDetection != nil {
		s.Rows = append(s.Rows, Row{
			Key:   "Detection",
			Value: shorten(data.CompletionDetection.Path),
			Note:  formatBytes(data.CompletionDetection.Size),
			Tone:  ToneMuted,
		})
	}
	if data.CompletionRegistry != nil && data.CompletionRegistry.Size > 0 {
		s.Rows = append(s.Rows, Row{
			Key:   "Registry",
			Value: shorten(data.CompletionRegistry.Path),
			Note:  formatBytes(data.CompletionRegistry.Size),
			Tone:  ToneMuted,
		})
	}

	return s
}

// countLabel renders "1 alias" / "9 aliases". The plural defaults to the
// singular with an "s".
func countLabel(n int, singular string, plural ...string) string {
	if n == 1 {
		return "1 " + singular
	}
	word := singular + "s"
	if len(plural) > 0 {
		word = plural[0]
	}
	return fmt.Sprintf("%d %s", n, word)
}

// firstWord returns the command a shell would actually run, arguments aside.
func firstWord(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return filepath.Base(fields[0])
}

// shorten replaces the home directory with ~ wherever it appears, including
// inside an alias command. Nothing else is abbreviated: a path shown here is
// a path the user can paste back.
func shorten(s string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || home == string(filepath.Separator) {
		return s
	}
	return strings.ReplaceAll(s, home, "~")
}

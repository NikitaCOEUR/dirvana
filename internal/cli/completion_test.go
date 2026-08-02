package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cobraTool answers the Cobra __complete protocol: it echoes one suggestion
// per known subcommand plus the trailing directive line.
const cobraTool = `#!/usr/bin/env bash
if [ "$1" != "__complete" ]; then
  exit 1
fi
printf "delete\tDelete a resource\n"
printf "apply\tApply a configuration\n"
printf "attach\tAttach to a container\n"
printf ":4\n"
`

// completionParams builds the params for the alias being completed
func (e *testEnv) completionParams(words []string, cword int) CompletionParams {
	return CompletionParams{
		CachePath: e.CachePath,
		AuthPath:  e.AuthPath,
		LogLevel:  "error",
		Words:     words,
		CWord:     cword,
	}
}

func TestCompletion_EmptyWords(t *testing.T) {
	env := newTestEnv(t)

	out := captureStdout(t, func() {
		assert.NoError(t, Completion(env.completionParams([]string{}, 0)))
	})
	assert.Empty(t, out)
}

func TestCompletion_NoContext(t *testing.T) {
	env := newTestEnv(t)

	// No config at all: nothing to complete, and no error either
	out := captureStdout(t, func() {
		assert.NoError(t, Completion(env.completionParams([]string{"anything"}, 0)))
	})
	assert.Empty(t, out)
}

func TestCompletion_UnauthorizedConfigIsIgnored(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	env.writeConfig(t, "aliases:\n  k: "+tool+"\n")
	// Deliberately not authorized

	out := captureStdout(t, func() {
		assert.NoError(t, Completion(env.completionParams([]string{"k", ""}, 1)))
	})
	assert.Empty(t, out)
}

func TestCompletion_AliasNotFound(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  other: echo other\n")
	env.allow(t)

	out := captureStdout(t, func() {
		assert.NoError(t, Completion(env.completionParams([]string{"nonexistent", ""}, 1)))
	})
	assert.Empty(t, out)
}

func TestCompletion_FunctionAliasHasNoSuggestions(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "functions:\n  myfunc: |\n    echo hello\n")
	env.allow(t)

	out := captureStdout(t, func() {
		assert.NoError(t, Completion(env.completionParams([]string{"myfunc", "arg1"}, 1)))
	})
	assert.Empty(t, out)
}

func TestCompletion_EmptyCommand(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  empty: \"\"\n")
	env.allow(t)

	out := captureStdout(t, func() {
		assert.NoError(t, Completion(env.completionParams([]string{"empty", ""}, 1)))
	})
	assert.Empty(t, out)
}

func TestCompletion_CommandNotFound(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(t, "aliases:\n  gone: dirvana-does-not-exist-anywhere\n")
	env.allow(t)

	out := captureStdout(t, func() {
		assert.NoError(t, Completion(env.completionParams([]string{"gone", "arg"}, 1)))
	})
	assert.Empty(t, out)
}

func TestCompletion_SuggestionsAreSortedWithDescriptions(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	env.writeConfig(t, "aliases:\n  k: "+tool+"\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", ""}, 1)))
	})

	lines := splitLines(out)
	require.Len(t, lines, 3)
	// Sorted by value, description kept after a tab
	assert.Equal(t, "apply\tApply a configuration", lines[0])
	assert.Equal(t, "attach\tAttach to a container", lines[1])
	assert.Equal(t, "delete\tDelete a resource", lines[2])
}

func TestCompletion_FiltersOnCurrentWord(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	env.writeConfig(t, "aliases:\n  k: "+tool+"\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", "a"}, 1)))
	})

	lines := splitLines(out)
	require.Len(t, lines, 2)
	assert.Equal(t, "apply\tApply a configuration", lines[0])
	assert.Equal(t, "attach\tAttach to a container", lines[1])
}

func TestCompletion_CompletionOverrideDrivesSuggestions(t *testing.T) {
	env := newTestEnv(t)
	// The alias runs a command with no completion support, but completion is
	// delegated to another tool
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	env.writeConfig(t, "aliases:\n  k:\n    command: echo\n    completion: "+tool+"\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", "de"}, 1)))
	})

	assert.Equal(t, []string{"delete\tDelete a resource"}, splitLines(out))
}

func TestCompletion_ToolWithoutSuggestions(t *testing.T) {
	env := newTestEnv(t)
	// A tool that speaks no completion protocol at all
	tool := env.mockTool(t, "mock-mute", "#!/usr/bin/env bash\nexit 1\n")
	env.writeConfig(t, "aliases:\n  m: "+tool+"\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"m", ""}, 1)))
	})

	assert.Empty(t, out)
}

func TestCompletion_SecondCallUsesDetectionCache(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	env.writeConfig(t, "aliases:\n  k: "+tool+"\n")
	env.allow(t)

	// First call detects the protocol and persists it
	first := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", "de"}, 1)))
	})
	require.FileExists(t, filepath.Join(env.Root, "completion-detection.json"))

	// Second call takes the cached-detection path, which skips the PATH
	// lookup, and must yield the same suggestions
	second := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", "de"}, 1)))
	})

	assert.Equal(t, []string{"delete\tDelete a resource"}, splitLines(first))
	assert.Equal(t, first, second)
}

func TestCompletion_PrefixMatchingNothing(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	env.writeConfig(t, "aliases:\n  k: "+tool+"\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", "zzz"}, 1)))
	})

	assert.Empty(t, out)
}

func TestCompletion_SuggestionsWithoutDescription(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-plain", `#!/usr/bin/env bash
if [ "$1" != "__complete" ]; then
  exit 1
fi
printf "beta\n"
printf "alpha\n"
printf ":4\n"
`)
	env.writeConfig(t, "aliases:\n  p: "+tool+"\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"p", ""}, 1)))
	})

	assert.Equal(t, []string{"alpha", "beta"}, splitLines(out))
}

func TestCompletion_CommandWithArgumentsUsesBaseCommand(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	// Extra arguments in the command must not break the lookup: only the
	// first field is the executable
	env.writeConfig(t, "aliases:\n  k: "+tool+" --context prod\n")
	env.allow(t)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", "at"}, 1)))
	})

	assert.Equal(t, []string{"attach\tAttach to a container"}, splitLines(out))
}

func TestCompletion_InheritsParentConfig(t *testing.T) {
	env := newTestEnv(t)
	tool := env.mockTool(t, "mock-cobra", cobraTool)
	env.writeConfig(t, "aliases:\n  k: "+tool+"\n")
	env.allow(t)

	child := filepath.Join(env.Dir, "child")
	env.writeConfigIn(t, child, "aliases:\n  extra: echo extra\n")
	env.allowDir(t, child)
	chdir(t, child)

	out := captureStdout(t, func() {
		require.NoError(t, Completion(env.completionParams([]string{"k", "de"}, 1)))
	})

	assert.Equal(t, []string{"delete\tDelete a resource"}, splitLines(out))
}

func TestResolveCompletionCommand(t *testing.T) {
	log := logger.New("error", nil)

	t.Run("no context", func(t *testing.T) {
		env := newTestEnv(t)

		_, _, err := resolveCompletionCommand("k", env.Dir, env.CachePath, env.AuthPath, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no dirvana context")
	})

	t.Run("unknown alias", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "aliases:\n  k: kubectl\n")
		env.allow(t)

		_, _, err := resolveCompletionCommand("unknown", env.Dir, env.CachePath, env.AuthPath, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a dirvana-managed alias")
	})

	t.Run("alias without override", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "aliases:\n  k: kubectl get pods\n")
		env.allow(t)

		command, completionCmd, err := resolveCompletionCommand("k", env.Dir, env.CachePath, env.AuthPath, log)
		require.NoError(t, err)
		assert.Equal(t, "kubectl get pods", command)
		assert.Equal(t, "kubectl get pods", completionCmd)
	})

	t.Run("alias with override", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "aliases:\n  k:\n    command: kubectl get pods\n    completion: kubectl\n")
		env.allow(t)

		command, completionCmd, err := resolveCompletionCommand("k", env.Dir, env.CachePath, env.AuthPath, log)
		require.NoError(t, err)
		assert.Equal(t, "kubectl get pods", command)
		assert.Equal(t, "kubectl", completionCmd)
	})

	t.Run("invalid cache path", func(t *testing.T) {
		env := newTestEnv(t)
		// A directory where the cache file is expected makes the engine fail
		_, _, err := resolveCompletionCommand("k", env.Dir, env.Dir, env.AuthPath, log)
		require.Error(t, err)
	})
}

func TestPrepareCompletionArgs(t *testing.T) {
	log := logger.New("error", nil)

	tests := []struct {
		name   string
		words  []string
		cword  int
		expect []string
	}{
		{
			name:   "alias alone gets an empty arg",
			words:  []string{"k"},
			cword:  0,
			expect: []string{""},
		},
		{
			name:   "word being completed is kept",
			words:  []string{"k", "get"},
			cword:  1,
			expect: []string{"get"},
		},
		{
			name:   "cword beyond the last word appends an empty one",
			words:  []string{"k", "get"},
			cword:  2,
			expect: []string{"get", ""},
		},
		{
			name:   "trailing empty word is not doubled",
			words:  []string{"k", "get", ""},
			cword:  3,
			expect: []string{"get", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := prepareCompletionArgs(CompletionParams{Words: tt.words, CWord: tt.cword}, log)
			assert.Equal(t, tt.expect, args)
		})
	}
}

func TestGetCurrentWord(t *testing.T) {
	tests := []struct {
		name   string
		words  []string
		cword  int
		expect string
	}{
		{name: "alias itself", words: []string{"k", "get"}, cword: 0, expect: ""},
		{name: "word being completed", words: []string{"k", "ge"}, cword: 1, expect: "ge"},
		{name: "beyond the last word", words: []string{"k", "get"}, cword: 5, expect: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, getCurrentWord(CompletionParams{Words: tt.words, CWord: tt.cword}))
		})
	}
}

// splitLines returns the non-empty lines of an output
func splitLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashCodeGenerator_Name(t *testing.T) {
	gen := NewCompletionGenerator(Bash)
	assert.Equal(t, "bash", gen.Name())
}

func TestBashCodeGenerator_GenerateCompletionFunction(t *testing.T) {
	gen := NewCompletionGenerator(Bash)
	aliasCommands := map[string]string{
		"k": "kubectl",
		"g": "git",
		"d": "docker",
	}

	lines := gen.GenerateCompletionFunction(aliasCommands)

	// Should have content
	assert.NotEmpty(t, lines)

	// Join to check full script
	script := strings.Join(lines, "\n")

	// Should contain bash shebang
	assert.Contains(t, script, "#!/usr/bin/env bash")

	// Should contain the completion function
	assert.Contains(t, script, "__dirvana_complete")

	// Should contain the register helper
	assert.Contains(t, script, "__dirvana_register_completion")

	// Should register completion for each alias with underlying command
	assert.Contains(t, script, "__dirvana_register_completion d docker")
	assert.Contains(t, script, "__dirvana_register_completion g git")
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")

	// Should have bash-specific features
	assert.Contains(t, script, "COMPREPLY")
	assert.Contains(t, script, "compgen")
}

func TestBashCodeGenerator_GenerateCompletionFunction_SingleAlias(t *testing.T) {
	gen := NewCompletionGenerator(Bash)
	aliasCommands := map[string]string{"k": "kubectl"}

	lines := gen.GenerateCompletionFunction(aliasCommands)
	script := strings.Join(lines, "\n")

	// Should register completion for the single alias
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")
}

func TestBashCodeGenerator_GenerateCompletionFunction_NoAliases(t *testing.T) {
	gen := NewCompletionGenerator(Bash)
	aliasCommands := map[string]string{}

	lines := gen.GenerateCompletionFunction(aliasCommands)
	script := strings.Join(lines, "\n")

	// Should still generate the function but without registration calls
	assert.Contains(t, script, "__dirvana_complete")
	assert.NotContains(t, script, "__dirvana_register_completion k")
}

func TestBashCodeGenerator_GenerateCompletionFunction_FunctionFallback(t *testing.T) {
	gen := NewCompletionGenerator(Bash)
	// Functions have empty underlying command
	aliasCommands := map[string]string{
		"k":      "kubectl",
		"myfunc": "",
	}

	lines := gen.GenerateCompletionFunction(aliasCommands)
	script := strings.Join(lines, "\n")

	// Alias with underlying command
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")
	// Function with no underlying command (empty second arg = dirvana fallback)
	assert.Contains(t, script, "__dirvana_register_completion myfunc ")
}

func TestZshCodeGenerator_Name(t *testing.T) {
	gen := NewCompletionGenerator(Zsh)
	assert.Equal(t, "zsh", gen.Name())
}

func TestZshCodeGenerator_GenerateCompletionFunction(t *testing.T) {
	gen := NewCompletionGenerator(Zsh)
	aliasCommands := map[string]string{
		"k": "kubectl",
		"g": "git",
	}

	lines := gen.GenerateCompletionFunction(aliasCommands)

	// Should have content
	assert.NotEmpty(t, lines)

	// Join to check full script
	script := strings.Join(lines, "\n")

	// Should contain zsh shebang
	assert.Contains(t, script, "#!/usr/bin/env zsh")

	// Should contain the completion function
	assert.Contains(t, script, "__dirvana_complete_zsh")

	// Should contain the register helper
	assert.Contains(t, script, "__dirvana_register_completion")

	// Should register completion for each alias with underlying command
	assert.Contains(t, script, "__dirvana_register_completion g git")
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")

	// Should have zsh-specific features
	assert.Contains(t, script, "_describe")
	assert.Contains(t, script, "CURRENT")
}

func TestZshCodeGenerator_GenerateCompletionFunction_SingleAlias(t *testing.T) {
	gen := NewCompletionGenerator(Zsh)
	aliasCommands := map[string]string{"k": "kubectl"}

	lines := gen.GenerateCompletionFunction(aliasCommands)
	script := strings.Join(lines, "\n")

	// Should register completion for the single alias
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")
}

func TestZshCodeGenerator_GenerateCompletionFunction_MultipleAliases(t *testing.T) {
	gen := NewCompletionGenerator(Zsh)
	aliasCommands := map[string]string{
		"k": "kubectl",
		"g": "git",
		"d": "docker",
	}

	lines := gen.GenerateCompletionFunction(aliasCommands)
	script := strings.Join(lines, "\n")

	// Should have separate registration for each alias
	assert.Contains(t, script, "__dirvana_register_completion d docker")
	assert.Contains(t, script, "__dirvana_register_completion g git")
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")

	// Should have the function defined once
	count := strings.Count(script, "__dirvana_complete_zsh()")
	assert.Equal(t, 1, count, "should have function definition once")
}

func TestZshCodeGenerator_GenerateCompletionFunction_FunctionFallback(t *testing.T) {
	gen := NewCompletionGenerator(Zsh)
	aliasCommands := map[string]string{
		"k":      "kubectl",
		"myfunc": "",
	}

	lines := gen.GenerateCompletionFunction(aliasCommands)
	script := strings.Join(lines, "\n")

	// Alias with underlying command
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")
	// Function with no underlying command
	assert.Contains(t, script, "__dirvana_register_completion myfunc ")
}

func TestMultiShellCodeGenerator_Name(t *testing.T) {
	gen := &MultiShellCodeGenerator{}
	assert.Equal(t, "multi", gen.Name())
}

func TestMultiShellCodeGenerator_GenerateCompletionFunction(t *testing.T) {
	gen := &MultiShellCodeGenerator{
		generators: []CodeGenerator{
			NewCompletionGenerator(Bash),
			NewCompletionGenerator(Zsh),
		},
	}
	aliasCommands := map[string]string{
		"k": "kubectl",
		"g": "git",
	}

	lines := gen.GenerateCompletionFunction(aliasCommands)

	// Should have content
	assert.NotEmpty(t, lines)

	// Join to check full script
	script := strings.Join(lines, "\n")

	// Should contain both bash and zsh shebangs
	assert.Contains(t, script, "#!/usr/bin/env bash")
	assert.Contains(t, script, "#!/usr/bin/env zsh")

	// Should contain both completion functions
	assert.Contains(t, script, "__dirvana_complete")
	assert.Contains(t, script, "__dirvana_complete_zsh")

	// Should have bash-specific features
	assert.Contains(t, script, "COMPREPLY")
	assert.Contains(t, script, "__dirvana_register_completion g git")
	assert.Contains(t, script, "__dirvana_register_completion k kubectl")

	// Should have zsh-specific features
	assert.Contains(t, script, "_describe")
}

func TestNewCompletionGenerator_Bash(t *testing.T) {
	gen := NewCompletionGenerator("bash")
	assert.NotNil(t, gen)
	assert.Equal(t, "bash", gen.Name())
}

func TestNewCompletionGenerator_Zsh(t *testing.T) {
	gen := NewCompletionGenerator("zsh")
	assert.NotNil(t, gen)
	assert.Equal(t, "zsh", gen.Name())
}

func TestNewCompletionGenerator_Fish(t *testing.T) {
	gen := NewCompletionGenerator("fish")
	assert.NotNil(t, gen)
	assert.Equal(t, "fish", gen.Name())

	// Verify it's actually a FishCodeGenerator
	_, ok := gen.(*FishCodeGenerator)
	assert.True(t, ok, "should return FishCodeGenerator")
}

func TestNewCompletionGenerator_Multi(t *testing.T) {
	// Any unknown shell should return multi-shell generator
	testCases := []string{"", "unknown", "both"}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			gen := NewCompletionGenerator(tc)
			assert.NotNil(t, gen)
			assert.Equal(t, "multi", gen.Name())

			// Verify it's actually a MultiShellCodeGenerator
			multiGen, ok := gen.(*MultiShellCodeGenerator)
			assert.True(t, ok, "should return MultiShellCodeGenerator")

			// Verify it has bash, zsh, and fish generators
			assert.Len(t, multiGen.generators, 3)
		})
	}
}

func TestNewCompletionGenerator_Integration(t *testing.T) {
	// Test that each generator can actually generate valid completion code
	testCases := []struct {
		shell         string
		aliasCommands map[string]string
	}{
		{"bash", map[string]string{"k": "kubectl", "g": "git"}},
		{"zsh", map[string]string{"k": "kubectl", "g": "git"}},
		{"fish", map[string]string{"k": "kubectl", "g": "git"}},
		{"multi", map[string]string{"k": "kubectl", "g": "git"}},
	}

	for _, tc := range testCases {
		t.Run(tc.shell, func(t *testing.T) {
			gen := NewCompletionGenerator(tc.shell)
			lines := gen.GenerateCompletionFunction(tc.aliasCommands)

			// Should generate non-empty output
			assert.NotEmpty(t, lines)

			script := strings.Join(lines, "\n")

			// Should contain shebang
			assert.Contains(t, script, "#!/usr/bin")

			// Should contain dirvana completion function
			assert.Contains(t, script, "dirvana_complete")

			// Should contain at least one alias
			for alias := range tc.aliasCommands {
				assert.Contains(t, script, alias)
			}
		})
	}
}

func TestGenerateHookCode_Bash(t *testing.T) {
	code, err := GenerateHookCode("bash", "dirvana")
	assert.NoError(t, err)
	assert.NotEmpty(t, code)

	// Should contain bash-specific features
	assert.Contains(t, code, "__dirvana_hook()")
	assert.Contains(t, code, "PROMPT_COMMAND")
	assert.Contains(t, code, "dirvana export")
	assert.Contains(t, code, "[[ ! -t 0 ]]", "should check stdin is terminal")

	// Should NOT contain zsh-specific features
	assert.NotContains(t, code, "add-zsh-hook")
	assert.NotContains(t, code, "autoload")
}

func TestGenerateHookCode_Zsh(t *testing.T) {
	code, err := GenerateHookCode("zsh", "dirvana")
	assert.NoError(t, err)
	assert.NotEmpty(t, code)

	// Should contain zsh-specific features
	assert.Contains(t, code, "__dirvana_hook()")
	assert.Contains(t, code, "autoload -U add-zsh-hook")
	assert.Contains(t, code, "add-zsh-hook chpwd")
	assert.Contains(t, code, "dirvana export")
	assert.Contains(t, code, "[[ ! -t 0 ]]", "should check stdin is terminal")

	// Should NOT contain bash-specific features
	assert.NotContains(t, code, "PROMPT_COMMAND")
}

func TestGenerateHookCode_DefaultToBash(t *testing.T) {
	// Unknown shell should default to bash
	code, err := GenerateHookCode("unknown", "dirvana")
	assert.NoError(t, err)
	assert.NotEmpty(t, code)

	// Should contain bash-specific features
	assert.Contains(t, code, "PROMPT_COMMAND")
}

func TestGenerateHookCode_BinaryPath(t *testing.T) {
	// Test with custom binary path
	code, err := GenerateHookCode("bash", "/usr/local/bin/dirvana")
	assert.NoError(t, err)
	assert.NotEmpty(t, code)

	// Should use the custom binary path
	assert.Contains(t, code, "/usr/local/bin/dirvana export")
}

func TestGenerateHookCode_MinimalDesign(t *testing.T) {
	// Both bash and zsh should generate minimal hooks
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			code, err := GenerateHookCode(shell, "dirvana")
			assert.NoError(t, err)

			// Should be minimal (less than 30 lines)
			lines := strings.Split(code, "\n")
			assert.Less(t, len(lines), 30, "hook should be minimal")

			// Should delegate to 'dirvana export'
			assert.Contains(t, code, "dirvana export")
			assert.Contains(t, code, "eval")

			// Should have stdin check (fast path for TUI apps)
			assert.Contains(t, code, "[[ ! -t 0 ]]")
		})
	}
}

func TestFishCodeGenerator_Name(t *testing.T) {
	gen := &FishCodeGenerator{}
	assert.Equal(t, "fish", gen.Name())
}

func TestFishCodeGenerator_GenerateCompletionFunction(t *testing.T) {
	gen := &FishCodeGenerator{}

	// Test with single alias that has underlying command
	aliasCommands := map[string]string{"k": "kubectl"}
	lines := gen.GenerateCompletionFunction(aliasCommands)
	assert.NotEmpty(t, lines)
	script := strings.Join(lines, "\n")
	assert.Contains(t, script, "function __dirvana_complete_fish")
	assert.Contains(t, script, "complete -c k -f")
	assert.Contains(t, script, "complete -c k -w kubectl")
	assert.Contains(t, script, "complete -c k -a '(__dirvana_complete_fish kubectl)'")

	// Test with multiple aliases
	aliasCommands = map[string]string{"k": "kubectl", "d": "docker"}
	lines = gen.GenerateCompletionFunction(aliasCommands)
	assert.NotEmpty(t, lines)
	script = strings.Join(lines, "\n")
	assert.Contains(t, script, "complete -c d -f")
	assert.Contains(t, script, "complete -c d -w docker")
	assert.Contains(t, script, "complete -c k -f")
	assert.Contains(t, script, "complete -c k -w kubectl")
}

func TestFishCodeGenerator_GenerateCompletionFunction_FunctionFallback(t *testing.T) {
	gen := &FishCodeGenerator{}

	// Function with no underlying command should not have -w
	aliasCommands := map[string]string{"myfunc": ""}
	lines := gen.GenerateCompletionFunction(aliasCommands)
	script := strings.Join(lines, "\n")

	assert.NotContains(t, script, "complete -c myfunc -w")
	assert.Contains(t, script, "complete -c myfunc -f")
	assert.Contains(t, script, "complete -c myfunc -a '(__dirvana_complete_fish)'")
}

func TestFishCodeGenerator_GenerateCompletionFunction_DefersToFishWhenCovered(t *testing.T) {
	gen := &FishCodeGenerator{}

	aliasCommands := map[string]string{"k": "kubectl"}
	script := strings.Join(gen.GenerateCompletionFunction(aliasCommands), "\n")

	// The wrapped command is handed to the completion function, which needs
	// it to tell the two cases apart
	assert.Contains(t, script, "__dirvana_complete_fish kubectl")
	assert.Contains(t, script, "underlying_cmd")

	// Asking dirvana costs a fork on every keypress, so it must stand down
	// whenever `complete -w` already forwards to real completions - and only
	// then, since a command fish knows nothing about gets no completion at
	// all without dirvana.
	assert.Contains(t, script, "function __dirvana_fish_covers")
	assert.Contains(t, script, `if test -n "$underlying_cmd"; and __dirvana_fish_covers $underlying_cmd`)

	// The probe must ask for long flags: file names never match them, so a
	// non-empty answer proves the command has completions of its own
	assert.Contains(t, script, `complete -C"$cmd --"`)

	// And it must be answered once per command, not on every keypress
	assert.Contains(t, script, "set -g $varname")
}

func TestGenerateHookCode_Fish(t *testing.T) {
	code, err := GenerateHookCode("fish", "dirvana")
	assert.NoError(t, err)
	assert.NotEmpty(t, code)

	// Fish hook should contain fish-specific syntax
	assert.Contains(t, code, "function")
	assert.Contains(t, code, "dirvana export")
}

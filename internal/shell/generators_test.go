package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashCodeGenerator_Name(t *testing.T) {
	gen := &BashCodeGenerator{}
	assert.Equal(t, "bash", gen.Name())
}

func TestBashCodeGenerator_GenerateCompletionFunction(t *testing.T) {
	gen := &BashCodeGenerator{}
	aliases := []string{"k", "g", "d"}

	lines := gen.GenerateCompletionFunction(aliases)

	// Should have content
	assert.NotEmpty(t, lines)

	// Join to check full script
	script := strings.Join(lines, "\n")

	// Should contain bash shebang
	assert.Contains(t, script, "#!/usr/bin/env bash")

	// Should contain the completion function
	assert.Contains(t, script, "__dirvana_complete")

	// Should register completion for all aliases in one command
	assert.Contains(t, script, "complete -o nosort -F __dirvana_complete k g d")

	// Should have bash-specific features
	assert.Contains(t, script, "COMPREPLY")
	assert.Contains(t, script, "compgen")
}

func TestBashCodeGenerator_GenerateCompletionFunction_SingleAlias(t *testing.T) {
	gen := &BashCodeGenerator{}
	aliases := []string{"kubectl"}

	lines := gen.GenerateCompletionFunction(aliases)
	script := strings.Join(lines, "\n")

	// Should register completion for the single alias
	assert.Contains(t, script, "complete -o nosort -F __dirvana_complete kubectl")
}

func TestBashCodeGenerator_GenerateCompletionFunction_NoAliases(t *testing.T) {
	gen := &BashCodeGenerator{}
	aliases := []string{}

	lines := gen.GenerateCompletionFunction(aliases)
	script := strings.Join(lines, "\n")

	// Should still generate the function but without aliases in complete command
	assert.Contains(t, script, "__dirvana_complete")
	assert.Contains(t, script, "complete -o nosort -F __dirvana_complete ")
}

func TestZshCodeGenerator_Name(t *testing.T) {
	gen := &ZshCodeGenerator{}
	assert.Equal(t, "zsh", gen.Name())
}

func TestZshCodeGenerator_GenerateCompletionFunction(t *testing.T) {
	gen := &ZshCodeGenerator{}
	aliases := []string{"k", "g"}

	lines := gen.GenerateCompletionFunction(aliases)

	// Should have content
	assert.NotEmpty(t, lines)

	// Join to check full script
	script := strings.Join(lines, "\n")

	// Should contain zsh shebang (appears multiple times, once per alias)
	assert.Contains(t, script, "#!/usr/bin/env zsh")

	// Should contain the completion function (appears multiple times)
	assert.Contains(t, script, "__dirvana_complete_zsh")

	// Should register completion for each alias separately
	assert.Contains(t, script, "compdef __dirvana_complete_zsh k")
	assert.Contains(t, script, "compdef __dirvana_complete_zsh g")

	// Should have zsh-specific features
	assert.Contains(t, script, "_describe")
	assert.Contains(t, script, "CURRENT")
}

func TestZshCodeGenerator_GenerateCompletionFunction_SingleAlias(t *testing.T) {
	gen := &ZshCodeGenerator{}
	aliases := []string{"kubectl"}

	lines := gen.GenerateCompletionFunction(aliases)
	script := strings.Join(lines, "\n")

	// Should register completion for the single alias
	assert.Contains(t, script, "compdef __dirvana_complete_zsh kubectl")
}

func TestZshCodeGenerator_GenerateCompletionFunction_MultipleAliases(t *testing.T) {
	gen := &ZshCodeGenerator{}
	aliases := []string{"k", "g", "d"}

	lines := gen.GenerateCompletionFunction(aliases)
	script := strings.Join(lines, "\n")

	// Should have separate compdef for each alias
	assert.Contains(t, script, "compdef __dirvana_complete_zsh k")
	assert.Contains(t, script, "compdef __dirvana_complete_zsh g")
	assert.Contains(t, script, "compdef __dirvana_complete_zsh d")

	// Should have the function repeated for each alias
	count := strings.Count(script, "__dirvana_complete_zsh()")
	assert.Equal(t, 3, count, "should have function definition for each alias")
}

func TestMultiShellCodeGenerator_Name(t *testing.T) {
	gen := &MultiShellCodeGenerator{}
	assert.Equal(t, "multi", gen.Name())
}

func TestMultiShellCodeGenerator_GenerateCompletionFunction(t *testing.T) {
	gen := &MultiShellCodeGenerator{
		generators: []CodeGenerator{
			&BashCodeGenerator{},
			&ZshCodeGenerator{},
		},
	}
	aliases := []string{"k", "g"}

	lines := gen.GenerateCompletionFunction(aliases)

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
	assert.Contains(t, script, "complete -o nosort -F __dirvana_complete k g")

	// Should have zsh-specific features
	assert.Contains(t, script, "_describe")
	assert.Contains(t, script, "compdef __dirvana_complete_zsh k")
	assert.Contains(t, script, "compdef __dirvana_complete_zsh g")
}

func TestNewCompletionGenerator_Bash(t *testing.T) {
	gen := NewCompletionGenerator("bash")
	assert.NotNil(t, gen)
	assert.Equal(t, "bash", gen.Name())

	// Verify it's actually a BashCodeGenerator
	_, ok := gen.(*BashCodeGenerator)
	assert.True(t, ok, "should return BashCodeGenerator")
}

func TestNewCompletionGenerator_Zsh(t *testing.T) {
	gen := NewCompletionGenerator("zsh")
	assert.NotNil(t, gen)
	assert.Equal(t, "zsh", gen.Name())

	// Verify it's actually a ZshCodeGenerator
	_, ok := gen.(*ZshCodeGenerator)
	assert.True(t, ok, "should return ZshCodeGenerator")
}

func TestNewCompletionGenerator_Multi(t *testing.T) {
	// Any unknown shell should return multi-shell generator
	testCases := []string{"", "unknown", "fish", "both"}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			gen := NewCompletionGenerator(tc)
			assert.NotNil(t, gen)
			assert.Equal(t, "multi", gen.Name())

			// Verify it's actually a MultiShellCodeGenerator
			multiGen, ok := gen.(*MultiShellCodeGenerator)
			assert.True(t, ok, "should return MultiShellCodeGenerator")

			// Verify it has both bash and zsh generators
			assert.Len(t, multiGen.generators, 2)
		})
	}
}

func TestNewCompletionGenerator_Integration(t *testing.T) {
	// Test that each generator can actually generate valid completion code
	testCases := []struct {
		shell   string
		aliases []string
	}{
		{"bash", []string{"k", "g"}},
		{"zsh", []string{"k", "g"}},
		{"multi", []string{"k", "g"}},
	}

	for _, tc := range testCases {
		t.Run(tc.shell, func(t *testing.T) {
			gen := NewCompletionGenerator(tc.shell)
			lines := gen.GenerateCompletionFunction(tc.aliases)

			// Should generate non-empty output
			assert.NotEmpty(t, lines)

			script := strings.Join(lines, "\n")

			// Should contain shebang
			assert.Contains(t, script, "#!/usr/bin")

			// Should contain dirvana completion function
			assert.Contains(t, script, "dirvana_complete")

			// Should contain at least one alias
			for _, alias := range tc.aliases {
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

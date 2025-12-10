package shell

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

const (
	shellBash = "bash"
	shellZsh  = "zsh"
	shellFish = "fish"
)

// CodeGenerator is an interface for shell-specific completion code generation
// Implementations generate shell code for bash, zsh, etc.
type CodeGenerator interface {
	// GenerateCompletionFunction generates shell-specific completion code for aliases
	GenerateCompletionFunction(aliases []string) []string
	// Name returns the shell name (bash, zsh, etc.)
	Name() string
}

// BashCodeGenerator generates bash-specific shell completion code
type BashCodeGenerator struct{}

// Name returns the shell name for bash
func (b *BashCodeGenerator) Name() string {
	return shellBash
}

// GenerateCompletionFunction generates bash-specific completion functions
func (b *BashCodeGenerator) GenerateCompletionFunction(aliases []string) []string {
	aliasStr := strings.Join(aliases, " ")
	script := fmt.Sprintf(bashTemplate, aliasStr)
	return strings.Split(script, "\n")
}

// ZshCodeGenerator generates zsh-specific shell completion code
type ZshCodeGenerator struct{}

// Name returns the shell name for zsh
func (z *ZshCodeGenerator) Name() string {
	return shellZsh
}

// GenerateCompletionFunction generates zsh-specific completion functions
func (z *ZshCodeGenerator) GenerateCompletionFunction(aliases []string) []string {
	var lines []string

	// Add the common completion function once
	lines = append(lines, strings.Split(zshFunctionTemplate, "\n")...)

	// Add completion registration for each alias
	for _, alias := range aliases {
		script := fmt.Sprintf(zshTemplate, alias)
		lines = append(lines, strings.Split(script, "\n")...)
	}

	return lines
}

// FishCodeGenerator generates fish-specific shell completion code
type FishCodeGenerator struct{}

// Name returns the shell name for fish
func (f *FishCodeGenerator) Name() string {
	return shellFish
}

// GenerateCompletionFunction generates fish-specific completion functions
func (f *FishCodeGenerator) GenerateCompletionFunction(aliases []string) []string {
	var lines []string

	// Add the common completion function once
	lines = append(lines, strings.Split(fishFunctionTemplate, "\n")...)

	// Add completion registration for each alias
	for _, alias := range aliases {
		script := fmt.Sprintf(fishTemplate, alias)
		lines = append(lines, strings.Split(script, "\n")...)
	}

	return lines
}

// MultiShellCodeGenerator generates completion code for multiple shells
type MultiShellCodeGenerator struct {
	generators []CodeGenerator
}

// Name returns the shell name for multi-shell generator
func (m *MultiShellCodeGenerator) Name() string {
	return "multi"
}

// GenerateCompletionFunction generates completion functions for all configured shells
func (m *MultiShellCodeGenerator) GenerateCompletionFunction(aliases []string) []string {
	var lines []string
	for _, gen := range m.generators {
		lines = append(lines, gen.GenerateCompletionFunction(aliases)...)
	}
	return lines
}

// NewCompletionGenerator creates appropriate shell code generator for the given shell type
func NewCompletionGenerator(shell string) CodeGenerator {
	switch shell {
	case shellBash:
		return &BashCodeGenerator{}
	case shellZsh:
		return &ZshCodeGenerator{}
	case shellFish:
		return &FishCodeGenerator{}
	default:
		// All shells
		return &MultiShellCodeGenerator{
			generators: []CodeGenerator{
				&BashCodeGenerator{},
				&ZshCodeGenerator{},
				&FishCodeGenerator{},
			},
		}
	}
}

// GenerateHookCode generates shell hook code from embedded templates
func GenerateHookCode(shell, binaryPath string) (string, error) {
	var tmpl string
	switch shell {
	case shellBash:
		tmpl = bashHookTemplate
	case shellZsh:
		tmpl = zshHookTemplate
	case shellFish:
		tmpl = fishHookTemplate
	default:
		tmpl = bashHookTemplate // Default to bash
	}

	// Parse template
	t, err := template.New("hook").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse hook template: %w", err)
	}

	// Execute template with binary path
	var buf bytes.Buffer
	data := struct {
		BinaryPath string
	}{
		BinaryPath: binaryPath,
	}
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute hook template: %w", err)
	}

	return buf.String(), nil
}

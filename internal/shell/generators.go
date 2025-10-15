package shell

import (
	"fmt"
	"strings"
)

const (
	shellBash = "bash"
	shellZsh  = "zsh"
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
	for _, alias := range aliases {
		script := fmt.Sprintf(zshTemplate, alias)
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
	case "bash":
		return &BashCodeGenerator{}
	case "zsh":
		return &ZshCodeGenerator{}
	default:
		// Both shells
		return &MultiShellCodeGenerator{
			generators: []CodeGenerator{
				&BashCodeGenerator{},
				&ZshCodeGenerator{},
			},
		}
	}
}

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
	// GenerateCompletionFunction generates shell-specific completion code for aliases.
	// aliasCommands maps alias name -> underlying command (empty string for functions).
	GenerateCompletionFunction(aliasCommands map[string]string) []string
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
func (b *BashCodeGenerator) GenerateCompletionFunction(aliasCommands map[string]string) []string {
	var lines []string
	lines = append(lines, strings.Split(bashTemplate, "\n")...)
	for _, alias := range sortedKeys(aliasCommands) {
		lines = append(lines, fmt.Sprintf("__dirvana_register_completion %s %s", alias, aliasCommands[alias]))
	}
	return lines
}

// ZshCodeGenerator generates zsh-specific shell completion code
type ZshCodeGenerator struct{}

// Name returns the shell name for zsh
func (z *ZshCodeGenerator) Name() string {
	return shellZsh
}

// GenerateCompletionFunction generates zsh-specific completion functions
func (z *ZshCodeGenerator) GenerateCompletionFunction(aliasCommands map[string]string) []string {
	var lines []string
	lines = append(lines, strings.Split(zshFunctionTemplate, "\n")...)
	for _, alias := range sortedKeys(aliasCommands) {
		lines = append(lines, fmt.Sprintf("__dirvana_register_completion %s %s", alias, aliasCommands[alias]))
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
func (f *FishCodeGenerator) GenerateCompletionFunction(aliasCommands map[string]string) []string {
	var lines []string
	lines = append(lines, strings.Split(fishFunctionTemplate, "\n")...)
	for _, alias := range sortedKeys(aliasCommands) {
		cmd := aliasCommands[alias]
		// Standalone -f globally disables file completions for this command.
		// Without this, -w can inherit file completion behavior from the wrapped command.
		lines = append(lines, fmt.Sprintf("complete -c %s -f", alias))
		if cmd != "" {
			// Use fish's native wrapping only. complete -w triggers fish's
			// autoloading and provides zero-cost in-memory completion delegation.
			// We intentionally don't add a dirvana fallback (-a) here because
			// fish evaluates ALL completion providers on every TAB, and the
			// command substitution fork for the fallback adds ~5-10ms overhead
			// even when it returns empty (which it does when native completions exist).
			lines = append(lines, fmt.Sprintf("complete -c %s -w %s", alias, cmd))
		} else {
			lines = append(lines, fmt.Sprintf("complete -c %s -a '(__dirvana_complete_fish)'", alias))
		}
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
func (m *MultiShellCodeGenerator) GenerateCompletionFunction(aliasCommands map[string]string) []string {
	var lines []string
	for _, gen := range m.generators {
		lines = append(lines, gen.GenerateCompletionFunction(aliasCommands)...)
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

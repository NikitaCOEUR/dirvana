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

// registrationGenerator generates completion code for shells whose per-alias
// registration is a plain __dirvana_register_completion call (bash and zsh);
// only the embedded template differs between them.
type registrationGenerator struct {
	shell    string
	template string
}

// Name returns the shell name
func (g *registrationGenerator) Name() string {
	return g.shell
}

// GenerateCompletionFunction emits the shell template followed by one
// registration line per alias
func (g *registrationGenerator) GenerateCompletionFunction(aliasCommands map[string]string) []string {
	lines := strings.Split(g.template, "\n")
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
			lines = append(lines, fmt.Sprintf("complete -c %s -w %s", alias, cmd))
			lines = append(lines, fmt.Sprintf("complete -c %s -a '(__dirvana_complete_fish %s)'", alias, cmd))
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
		return &registrationGenerator{shell: shellBash, template: bashTemplate}
	case shellZsh:
		return &registrationGenerator{shell: shellZsh, template: zshFunctionTemplate}
	case shellFish:
		return &FishCodeGenerator{}
	default:
		// All shells
		return &MultiShellCodeGenerator{
			generators: []CodeGenerator{
				NewCompletionGenerator(shellBash),
				NewCompletionGenerator(shellZsh),
				NewCompletionGenerator(shellFish),
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

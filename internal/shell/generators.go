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
	var lines []string

	lines = append(lines, "__dirvana_complete() {")
	lines = append(lines, "  local cur prev words cword")
	lines = append(lines, "  _get_comp_words_by_ref -n : cur prev words cword 2>/dev/null || {")
	lines = append(lines, "    cur=\"${COMP_WORDS[COMP_CWORD]}\"")
	lines = append(lines, "    prev=\"${COMP_WORDS[COMP_CWORD-1]}\"")
	lines = append(lines, "    words=(\"${COMP_WORDS[@]}\")")
	lines = append(lines, "    cword=$COMP_CWORD")
	lines = append(lines, "  }")
	lines = append(lines, "")
	lines = append(lines, "  # Let dirvana handle ALL completion logic (fuzzy matching, filtering, etc.)")
	lines = append(lines, "  local IFS=$'\\n'")
	lines = append(lines, "  local suggestions")
	lines = append(lines, "  suggestions=($(DIRVANA_COMP_CWORD=$cword dirvana completion -- \"${words[@]}\" 2>/dev/null))")
	lines = append(lines, "")
	lines = append(lines, "  if [ ${#suggestions[@]} -gt 0 ]; then")
	lines = append(lines, "    COMPREPLY=(\"${suggestions[@]}\")")
	lines = append(lines, "")
	lines = append(lines, "    # Format descriptions like kubectl does (only if multiple suggestions)")
	lines = append(lines, "    if [ ${#COMPREPLY[@]} -gt 1 ]; then")
	lines = append(lines, "      __dirvana_format_descriptions")
	lines = append(lines, "    else")
	lines = append(lines, "      # Single suggestion: strip description directly without formatting overhead")
	lines = append(lines, "      local result=\"${COMPREPLY[0]}\"")
	lines = append(lines, "      result=\"${result%%$'\\t'*}\"  # Remove tab and description")
	lines = append(lines, "      COMPREPLY[0]=\"$result\"")
	lines = append(lines, "    fi")
	lines = append(lines, "  else")
	lines = append(lines, "    # Fallback to file completion")
	lines = append(lines, "    COMPREPLY=($(compgen -f -- \"$cur\"))")
	lines = append(lines, "  fi")
	lines = append(lines, "}")
	lines = append(lines, "")
	lines = append(lines, "# Format completion descriptions (value\\tdesc -> value  (desc))")
	lines = append(lines, "__dirvana_format_descriptions() {")
	lines = append(lines, "  local tab=$'\\t'")
	lines = append(lines, "  local comp desc maxdesclength longest=0")
	lines = append(lines, "  local i ci")
	lines = append(lines, "")
	lines = append(lines, "  # Find longest completion for alignment")
	lines = append(lines, "  for ci in \"${!COMPREPLY[@]}\"; do")
	lines = append(lines, "    comp=\"${COMPREPLY[ci]%%$tab*}\"")
	lines = append(lines, "    if ((${#comp} > longest)); then")
	lines = append(lines, "      longest=${#comp}")
	lines = append(lines, "    fi")
	lines = append(lines, "  done")
	lines = append(lines, "")
	lines = append(lines, "  # Format each completion with description")
	lines = append(lines, "  for ci in \"${!COMPREPLY[@]}\"; do")
	lines = append(lines, "    comp=\"${COMPREPLY[ci]}\"")
	lines = append(lines, "    if [[ \"$comp\" == *\"$tab\"* ]]; then")
	lines = append(lines, "      desc=\"${comp#*$tab}\"")
	lines = append(lines, "      comp=\"${comp%%$tab*}\"")
	lines = append(lines, "")
	lines = append(lines, "      # Calculate max description length")
	lines = append(lines, "      maxdesclength=$((COLUMNS - longest - 4))")
	lines = append(lines, "      if ((maxdesclength > 8)); then")
	lines = append(lines, "        # Pad completion to align descriptions")
	lines = append(lines, "        for ((i = ${#comp}; i < longest; i++)); do")
	lines = append(lines, "          comp+=\" \"")
	lines = append(lines, "        done")
	lines = append(lines, "      else")
	lines = append(lines, "        maxdesclength=$((COLUMNS - ${#comp} - 4))")
	lines = append(lines, "      fi")
	lines = append(lines, "")
	lines = append(lines, "      # Truncate description if too long")
	lines = append(lines, "      if ((maxdesclength > 0)); then")
	lines = append(lines, "        if ((${#desc} > maxdesclength)); then")
	lines = append(lines, "          desc=\"${desc:0:$((maxdesclength - 1))}…\"")
	lines = append(lines, "        fi")
	lines = append(lines, "        comp+=\"  ($desc)\"")
	lines = append(lines, "      fi")
	lines = append(lines, "")
	lines = append(lines, "      COMPREPLY[ci]=\"$comp\"")
	lines = append(lines, "    fi")
	lines = append(lines, "  done")
	lines = append(lines, "}")
	lines = append(lines, "")

	// Register bash completion for all aliases
	aliasStr := strings.Join(aliases, " ")
	lines = append(lines, fmt.Sprintf("complete -o nosort -F __dirvana_complete %s 2>/dev/null || true", aliasStr))
	lines = append(lines, "")

	return lines
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

	lines = append(lines, "# Zsh completions")
	lines = append(lines, "__dirvana_complete_zsh() {")
	lines = append(lines, "  local -a completions")
	lines = append(lines, "  local -a suggestions")
	lines = append(lines, "  local -a words_array")
	lines = append(lines, "  local cword")
	lines = append(lines, "  ")
	lines = append(lines, "  # In zsh, CURRENT is 1-based and points to the word being completed")
	lines = append(lines, "  # For dirvana completion, we need COMP_CWORD to be 0-based and point to the position where we want completions")
	lines = append(lines, "  #")
	lines = append(lines, "  # Examples:")
	lines = append(lines, "  #   \"k <TAB>\"     -> words=[\"k\"], CURRENT=2 -> we want cword=1 (completing after k)")
	lines = append(lines, "  #   \"k ann<TAB>\"  -> words=[\"k\",\"ann\"], CURRENT=2 -> we want cword=1 (completing word at position 1)")
	lines = append(lines, "  #   \"k get <TAB>\" -> words=[\"k\",\"get\"], CURRENT=3 -> we want cword=2 (completing after get)")
	lines = append(lines, "")
	lines = append(lines, "  words_array=(\"${words[@]}\")")
	lines = append(lines, "")
	lines = append(lines, "  # CURRENT points to the word being completed (1-based)")
	lines = append(lines, "  # We convert to 0-based for COMP_CWORD")
	lines = append(lines, "  cword=$((CURRENT - 1))")
	lines = append(lines, "")
	lines = append(lines, "  # If cursor is after all words (CURRENT > number of words), we're completing a new word")
	lines = append(lines, "  # Add an empty word to the array for completion")
	lines = append(lines, "  if (( CURRENT > ${#words[@]} )); then")
	lines = append(lines, "    words_array+=(\"\")")
	lines = append(lines, "  fi")
	lines = append(lines, "  ")
	lines = append(lines, "  # Call dirvana completion with all words from command line")
	lines = append(lines, "  local IFS=$'\\n'")
	lines = append(lines, "  suggestions=(${(f)\"$(DIRVANA_COMP_CWORD=$cword dirvana completion -- \"${words_array[@]}\" 2>/dev/null)\"})")
	lines = append(lines, "  ")
	lines = append(lines, "  # Check if we got any suggestions")
	lines = append(lines, "  if (( ${#suggestions[@]} == 0 )); then")
	lines = append(lines, "    # Fallback to file completion")
	lines = append(lines, "    _files")
	lines = append(lines, "    return 0")
	lines = append(lines, "  fi")
	lines = append(lines, "  ")
	lines = append(lines, "  # Parse suggestions (format: value\\tdescription)")
	lines = append(lines, "  # We use two arrays: one for values, one for descriptions")
	lines = append(lines, "  local -a values")
	lines = append(lines, "  local -a descriptions")
	lines = append(lines, "  local suggestion value desc")
	lines = append(lines, "  ")
	lines = append(lines, "  for suggestion in \"${suggestions[@]}\"; do")
	lines = append(lines, "    if [[ \"$suggestion\" == *$'\\t'* ]]; then")
	lines = append(lines, "      value=\"${suggestion%%$'\\t'*}\"")
	lines = append(lines, "      desc=\"${suggestion#*$'\\t'}\"")
	lines = append(lines, "      values+=(\"${value}\")")
	lines = append(lines, "      descriptions+=(\"${desc}\")")
	lines = append(lines, "      completions+=(\"${value}:${desc}\")")
	lines = append(lines, "    else")
	lines = append(lines, "      values+=(\"${suggestion}\")")
	lines = append(lines, "      descriptions+=(\"\")")
	lines = append(lines, "      completions+=(\"${suggestion}\")")
	lines = append(lines, "    fi")
	lines = append(lines, "  done")
	lines = append(lines, "  ")
	lines = append(lines, "  # Use _describe which automatically handles prefix matching in zsh")
	lines = append(lines, "  # The -V option disables sorting to preserve kubectl's order")
	lines = append(lines, "  if _describe -V 'completions' completions; then")
	lines = append(lines, "    return 0")
	lines = append(lines, "  fi")
	lines = append(lines, "  ")
	lines = append(lines, "  # If _describe didn't find matches, return error to trigger other matchers")
	lines = append(lines, "  return 1")
	lines = append(lines, "}")
	lines = append(lines, "")

	for _, alias := range aliases {
		lines = append(lines, fmt.Sprintf("compdef __dirvana_complete_zsh %s 2>/dev/null || true", alias))
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

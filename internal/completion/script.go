package completion

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ScriptCompleter handles tools that use standard bash completion scripts
// This is used by git, docker, systemctl, and many other Unix tools
// These tools have completion scripts in /usr/share/bash-completion/completions/
// or /etc/bash_completion.d/
type ScriptCompleter struct{}

// NewScriptCompleter creates a new script-based completer
func NewScriptCompleter() *ScriptCompleter {
	return &ScriptCompleter{}
}

// completionScriptPaths returns possible locations for bash completion scripts
func completionScriptPaths(tool string) []string {
	return []string{
		filepath.Join("/usr/share/bash-completion/completions", tool),
		filepath.Join("/usr/local/share/bash-completion/completions", tool),
		filepath.Join("/etc/bash_completion.d", tool),
		// Homebrew on macOS
		filepath.Join("/usr/local/etc/bash_completion.d", tool),
		filepath.Join("/opt/homebrew/etc/bash_completion.d", tool),
	}
}

// findCompletionScript finds the bash completion script for a tool
func findCompletionScript(tool string) string {
	for _, path := range completionScriptPaths(tool) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// Supports checks if the tool has a bash completion script available
func (s *ScriptCompleter) Supports(tool string, _ []string) bool {
	return findCompletionScript(tool) != ""
}

// Complete uses bash completion scripts to get suggestions
// It sources the completion script and calls the completion function
func (s *ScriptCompleter) Complete(tool string, args []string) ([]Suggestion, error) {
	scriptPath := findCompletionScript(tool)
	if scriptPath == "" {
		return nil, fmt.Errorf("no completion script found for %s", tool)
	}

	// Build the command line as bash would see it
	// COMP_WORDS is an array: (tool arg1 arg2 ...)
	// COMP_CWORD is the index of the word being completed
	compWords := append([]string{tool}, args...)
	compCword := len(compWords) - 1

	// If the last argument is empty, we're completing a new word
	if len(args) > 0 && args[len(args)-1] == "" {
		compCword = len(compWords) - 1
	}

	// Create a bash script that:
	// 1. Sources the completion script
	// 2. Calls the completion function
	// 3. Outputs COMPREPLY
	bashScript := fmt.Sprintf(`
set -e
# Source bash_completion framework if available
if [ -f /usr/share/bash-completion/bash_completion ]; then
    source /usr/share/bash-completion/bash_completion
elif [ -f /etc/bash_completion ]; then
    source /etc/bash_completion
fi

# Source the tool's completion script
source %s 2>/dev/null || exit 1

# Set up completion variables
COMP_WORDS=(%s)
COMP_CWORD=%d
COMP_LINE="%s"
COMP_POINT=${#COMP_LINE}

# Many completion scripts expect lowercase variables
words=("${COMP_WORDS[@]}")
cword=$COMP_CWORD
cur="${COMP_WORDS[COMP_CWORD]}"
prev="${COMP_WORDS[COMP_CWORD-1]}"

# Call the completion function
# The function name is usually _<tool> (e.g., _git, _docker)
# But can also be __<tool>_main or other variations
if declare -F __%s_main >/dev/null 2>&1; then
    __%s_main
elif declare -F _%s >/dev/null 2>&1; then
    _%s
elif declare -F __%s >/dev/null 2>&1; then
    __%s
else
    # Try to find any function with the tool name
    func=$(declare -F | grep -E "_(_%s|%s)" | head -1 | awk '{print $3}')
    if [ -n "$func" ]; then
        $func
    else
        exit 1
    fi
fi

# Output completions (one per line)
for completion in "${COMPREPLY[@]}"; do
    echo "$completion"
done
`,
		scriptPath,
		strings.Join(escapeShellWords(compWords), " "),
		compCword,
		strings.Join(compWords, " "),
		tool, tool, // __%s_main
		tool, tool, // _%s
		tool, tool, // __%s
		tool, tool, // grep pattern
	)

	// Execute the bash script
	cmd := exec.Command("bash", "-c", bashScript)
	cmd.Env = os.Environ()

	// Debug: uncomment to see the generated script
	// fmt.Fprintf(os.Stderr, "=== Bash script for %s with args %v ===\n%s\n===\n", tool, args, bashScript)

	output, err := cmd.Output()
	if err != nil {
		// If completion failed, it might be because the script doesn't support it
		// or the tool doesn't have completion for this context
		return nil, err
	}

	return parseScriptOutput(output), nil
}

// escapeShellWords escapes words for use in bash arrays
func escapeShellWords(words []string) []string {
	escaped := make([]string, len(words))
	for i, word := range words {
		// Escape single quotes and wrap in single quotes
		escaped[i] = "'" + strings.ReplaceAll(word, "'", "'\\''") + "'"
	}
	return escaped
}

// parseScriptOutput parses the output from bash completion scripts
// Format: one suggestion per line, may include descriptions separated by tab or space
func parseScriptOutput(output []byte) []Suggestion {
	suggestions := []Suggestion{} // Initialize as empty slice, not nil

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Some completions include descriptions separated by tab
		parts := strings.SplitN(line, "\t", 2)
		suggestion := Suggestion{
			Value: parts[0],
		}

		if len(parts) > 1 {
			suggestion.Description = parts[1]
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

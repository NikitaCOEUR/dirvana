package completion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScriptCompleter handles tools that use standard bash completion scripts
// This is used by git, docker, systemctl, and many other Unix tools
// These tools have completion scripts in /usr/share/bash-completion/completions/
// or /etc/bash_completion.d/, or can be auto-downloaded from the registry
type ScriptCompleter struct {
	cacheDir string
}

// NewScriptCompleter creates a new script-based completer
func NewScriptCompleter(cacheDir string) *ScriptCompleter {
	return &ScriptCompleter{cacheDir: cacheDir}
}

// completionScriptPaths returns possible locations for bash completion scripts
func (s *ScriptCompleter) completionScriptPaths(tool string) []string {
	paths := []string{
		filepath.Join("/usr/share/bash-completion/completions", tool),
		filepath.Join("/usr/local/share/bash-completion/completions", tool),
		filepath.Join("/etc/bash_completion.d", tool),
		// Homebrew on macOS
		filepath.Join("/usr/local/etc/bash_completion.d", tool),
		filepath.Join("/opt/homebrew/etc/bash_completion.d", tool),
	}

	// Add dirvana cache location (bash scripts only)
	// Note: We only use bash scripts as they work for all shells (bash, zsh, fish)
	if s.cacheDir != "" {
		paths = append(paths,
			GetCompletionScriptPath(s.cacheDir, tool, "bash"),
		)
	}

	return paths
}

// findCompletionScript finds the bash completion script for a tool
func (s *ScriptCompleter) findCompletionScript(tool string) string {
	for _, path := range s.completionScriptPaths(tool) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// Supports checks if the tool has a bash completion script available
// or can have one auto-installed
func (s *ScriptCompleter) Supports(tool string, _ []string) bool {
	// Check if script already exists
	if s.findCompletionScript(tool) != "" {
		return true
	}

	// Check if we can download from registry
	if s.cacheDir != "" {
		registry, err := LoadRegistry(s.cacheDir)
		if err == nil {
			if _, ok := registry.Tools[tool]; ok {
				return true
			}
		}
	}

	return false
}

// Complete uses bash completion scripts to get suggestions
// It sources the completion script and calls the completion function
func (s *ScriptCompleter) Complete(tool string, args []string) ([]Suggestion, error) {
	// Ensure script is available (find locally or download from registry)
	scriptPath, err := s.ensureScriptAvailable(tool)
	if err != nil {
		return nil, err
	}

	// Build bash completion script
	bashScript := s.buildBashCompletionScript(scriptPath, tool, args)

	// Execute the bash script
	output, err := s.executeBashScript(bashScript, tool)
	if err != nil {
		return nil, err
	}

	return parseScriptOutput(output), nil
}

// ensureScriptAvailable finds or downloads the completion script for a tool
func (s *ScriptCompleter) ensureScriptAvailable(tool string) (string, error) {
	scriptPath := s.findCompletionScript(tool)

	// If no script found, try to download from registry
	if scriptPath == "" && s.cacheDir != "" {
		registry, err := LoadRegistry(s.cacheDir)
		if err == nil {
			// Try to download for bash (default)
			if err := DownloadCompletionScript(s.cacheDir, tool, "bash", registry); err == nil {
				// Retry finding the script
				scriptPath = s.findCompletionScript(tool)
			}
		}

		if scriptPath == "" {
			return "", fmt.Errorf("no completion script found for %s", tool)
		}
	}

	return scriptPath, nil
}

// buildBashCompletionScript generates the bash script that will run the completion
func (s *ScriptCompleter) buildBashCompletionScript(scriptPath, tool string, args []string) string {
	// Build the command line as bash would see it
	// COMP_WORDS is an array: (tool arg1 arg2 ...)
	// COMP_CWORD is the index of the word being completed
	compWords := append([]string{tool}, args...)
	compCword := len(compWords) - 1

	// If the last argument is empty, we're completing a new word
	if len(args) > 0 && args[len(args)-1] == "" {
		compCword = len(compWords) - 1
	}

	// Escape COMP_LINE properly to prevent injection
	// Use single quotes and escape any single quotes inside
	escapedCompLine := "'" + strings.ReplaceAll(strings.Join(compWords, " "), "'", "'\\''") + "'"

	// Create a bash script that:
	// 1. Sources the completion script
	// 2. Calls the completion function
	// 3. Outputs COMPREPLY
	return fmt.Sprintf(`
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
COMP_LINE=%s
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
		escapedCompLine, // Now properly escaped
		tool, tool, // __%s_main
		tool, tool, // _%s
		tool, tool, // __%s
		tool, tool, // grep pattern
	)
}

// executeBashScript executes the bash completion script with timeout
func (s *ScriptCompleter) executeBashScript(bashScript, tool string) ([]byte, error) {
	// Debug: uncomment to see the generated script
	// fmt.Fprintf(os.Stderr, "=== Bash script for %s ===\n%s\n===\n", tool, bashScript)

	ctx := context.Background()
	output, err := execWithTimeoutAndEnv(ctx, os.Environ(), "bash", "-c", bashScript)
	if err != nil {
		return nil, fmt.Errorf("completion script failed for %s: %w", tool, err)
	}

	return output, nil
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
	// Use common parser with description support
	return parseCompletionOutput(output, true)
}

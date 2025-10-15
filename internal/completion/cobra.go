package completion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Cobra shell completion directives (from spf13/cobra)
const (
	ShellCompDirectiveError         = 1  // Error occurred
	ShellCompDirectiveNoSpace       = 2  // Don't add space after completion
	ShellCompDirectiveNoFileComp    = 4  // Don't suggest files
	ShellCompDirectiveFilterFileExt = 8  // Filter files by extension (suggestions are extensions)
	ShellCompDirectiveFilterDirs    = 16 // Only show directories
	ShellCompDirectiveKeepOrder     = 32 // Keep completion order
)

// CobraCompleter handles completion for Cobra-based CLIs (kubectl, helm, etc.)
type CobraCompleter struct{}

// NewCobraCompleter creates a new Cobra completer
func NewCobraCompleter() *CobraCompleter {
	return &CobraCompleter{}
}

// Supports checks if the tool supports Cobra's __complete API
// We verify by checking for Cobra's directive format in the output
func (c *CobraCompleter) Supports(tool string, _ []string) bool {
	// Try calling tool __complete with empty arg (with timeout)
	ctx := context.Background()
	output, err := execWithTimeout(ctx, tool, "__complete", "")

	// If command failed or returned nothing, doesn't support
	if err != nil || len(output) == 0 {
		return false
	}

	// Check if output contains a Cobra directive (line starting with ":")
	// Cobra always outputs a directive like ":4" or ":0" at the end
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 1 && line[0] == ':' {
			// Verify it looks like a number after the colon
			if _, err := fmt.Sscanf(line, ":%d", new(int)); err == nil {
				return true
			}
		}
	}

	return false
}

// Complete executes the tool's __complete command and parses the output
func (c *CobraCompleter) Complete(tool string, args []string) ([]Suggestion, error) {
	// Build the __complete command
	// tool __complete <args...>
	completeArgs := append([]string{"__complete"}, args...)

	ctx := context.Background()
	output, err := execWithTimeout(ctx, tool, completeArgs...)
	if err != nil {
		return nil, err
	}

	suggestions, directive := parseCobraOutput(output)

	// Handle directives that require file/directory completion
	if directive&ShellCompDirectiveFilterFileExt != 0 {
		// Suggestions are file extensions, list files matching those extensions
		return c.completeFilesWithExtensions(suggestions, args)
	}

	if directive&ShellCompDirectiveFilterDirs != 0 {
		// Only show directories
		return c.completeDirectories(args)
	}

	return suggestions, nil
}

// parseCobraOutput parses Cobra completion output format:
// - Lines with format: "value\tdescription"
// - Lines starting with ":" are directives
// - Empty lines are ignored
// Returns suggestions and the directive value
func parseCobraOutput(output []byte) ([]Suggestion, int) {
	var filteredLines []string
	directive := 0
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse Cobra directive (format: ":number")
		if strings.HasPrefix(line, ":") {
			if d, err := strconv.Atoi(strings.TrimPrefix(line, ":")); err == nil {
				directive = d
			}
			continue
		}

		// Skip help messages
		if strings.Contains(line, "Completion ended") || strings.Contains(line, "ShellCompDirective") {
			continue
		}

		filteredLines = append(filteredLines, line)
	}

	// Use common parser with description support
	filteredOutput := []byte(strings.Join(filteredLines, "\n"))
	return parseCompletionOutput(filteredOutput, true), directive
}

// completeFilesWithExtensions lists files matching the given extensions
// extensions are provided as suggestions (e.g., "json", "yaml", "yml")
func (c *CobraCompleter) completeFilesWithExtensions(extensionSuggestions []Suggestion, args []string) ([]Suggestion, error) {
	// Extract extensions from suggestions
	var extensions []string
	for _, s := range extensionSuggestions {
		ext := s.Value
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extensions = append(extensions, ext)
	}

	// Determine the directory and prefix to search
	prefix := ""
	searchDir := "."
	if len(args) > 0 {
		lastArg := args[len(args)-1]
		if lastArg != "" {
			dir := filepath.Dir(lastArg)
			base := filepath.Base(lastArg)

			// If lastArg ends with /, it's a directory
			if strings.HasSuffix(lastArg, "/") {
				searchDir = lastArg
				prefix = ""
			} else {
				searchDir = dir
				prefix = base
			}
		}
	}

	// List files in directory
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return []Suggestion{}, nil // Return empty on error, not an error
	}

	var suggestions []Suggestion
	for _, entry := range entries {
		name := entry.Name()

		// Filter by prefix
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}

		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		// For directories, always include them (to navigate)
		if entry.IsDir() {
			fullPath := filepath.Join(searchDir, name)
			if searchDir == "." {
				fullPath = name
			}
			suggestions = append(suggestions, Suggestion{
				Value:       fullPath + "/",
				Description: "",
			})
			continue
		}

		// For files, check if extension matches
		ext := filepath.Ext(name)
		matched := false
		for _, allowedExt := range extensions {
			if ext == allowedExt {
				matched = true
				break
			}
		}

		if matched {
			fullPath := filepath.Join(searchDir, name)
			if searchDir == "." {
				fullPath = name
			}
			suggestions = append(suggestions, Suggestion{
				Value:       fullPath,
				Description: "",
			})
		}
	}

	return suggestions, nil
}

// completeDirectories lists only directories
func (c *CobraCompleter) completeDirectories(args []string) ([]Suggestion, error) {
	// Determine the directory and prefix to search
	prefix := ""
	searchDir := "."
	if len(args) > 0 {
		lastArg := args[len(args)-1]
		if lastArg != "" {
			dir := filepath.Dir(lastArg)
			base := filepath.Base(lastArg)

			// If lastArg ends with /, it's a directory
			if strings.HasSuffix(lastArg, "/") {
				searchDir = lastArg
				prefix = ""
			} else {
				searchDir = dir
				prefix = base
			}
		}
	}

	// List directories
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return []Suggestion{}, nil // Return empty on error, not an error
	}

	var suggestions []Suggestion
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Filter by prefix
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(searchDir, name)
		if searchDir == "." {
			fullPath = name
		}
		suggestions = append(suggestions, Suggestion{
			Value:       fullPath + "/",
			Description: "",
		})
	}

	return suggestions, nil
}

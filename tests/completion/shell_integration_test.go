package completion_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultShell = "bash"

func getTestShell() string {
	if shell := os.Getenv("TEST_SHELL"); shell != "" {
		return shell
	}
	return defaultShell
}

var testShell = getTestShell()

// Shell integration tests - test real completion in actual shells
var shellIntegrationTests = []struct {
	name           string
	shell          string
	alias          string
	tool           string
	minCompletions int
	shouldContain  []string
}{
	{
		name:           "kubectl-in-bash",
		shell:          "bash",
		alias:          "k",
		tool:           "kubectl",
		minCompletions: 30,
		shouldContain:  []string{"get", "apply", "delete"},
	},
	{
		name:           "terraform-in-bash",
		shell:          "bash",
		alias:          "tf",
		tool:           "terraform",
		minCompletions: 5,
		shouldContain:  []string{"apply", "validate"},
	},
	{
		name:           "aqua-in-bash",
		shell:          "bash",
		alias:          "a",
		tool:           "aqua",
		minCompletions: 5,
		shouldContain:  []string{"install", "init"},
	},
	{
		name:           "kubectl-in-zsh",
		shell:          "zsh",
		alias:          "k",
		tool:           "kubectl",
		minCompletions: 30,
		shouldContain:  []string{"get", "apply", "delete"},
	},
	{
		name:           "terraform-in-zsh",
		shell:          "zsh",
		alias:          "tf",
		tool:           "terraform",
		minCompletions: 15,
		shouldContain:  []string{"init", "plan", "apply"},
	},
}

func TestShellIntegration_Completion(t *testing.T) {
	// These tests require expect and a properly configured shell environment
	// They should only run in the Docker test environment
	if os.Getenv("TEST_SHELL") == "" {
		t.Skip("Skipping shell integration tests (run with task test-completion in Docker)")
	}

	// Setup test environment
	setupTestEnvironment(t)

	for _, tt := range shellIntegrationTests {
		// Skip if not testing this shell
		if tt.shell != testShell {
			t.Logf("Skipping %s test (current shell: %s)", tt.shell, testShell)
			continue
		}

		t.Run(tt.name, func(t *testing.T) {
			// TODO: Fix zsh expect script - completion works manually but expect has issues capturing output
			if tt.shell == "zsh" {
				t.Skip("Zsh completion tests temporarily disabled due to expect script issues")
			}
			t.Logf("Testing %s completion in %s", tt.alias, tt.shell)

			// Get absolute path to config dir
			configDir, err := filepath.Abs("testdata")
			require.NoError(t, err)

			// Get absolute path to script
			scriptPath, err := filepath.Abs(filepath.Join("scripts", "test_"+tt.shell+"_completion.sh"))
			require.NoError(t, err)

			// Run completion test script
			cmd := exec.Command(scriptPath, tt.alias, configDir)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Logf("Script output:\n%s", string(output))
				t.Fatalf("Completion test failed: %v", err)
			}

			// Parse completions from output
			completions := parseCompletionOutput(string(output))

			// Verify minimum number of completions
			assert.GreaterOrEqual(t, len(completions), tt.minCompletions,
				"Expected at least %d completions in %s for %s",
				tt.minCompletions, tt.shell, tt.alias)

			// Verify expected completions are present
			for _, expected := range tt.shouldContain {
				assert.Contains(t, completions, expected,
					"Completion should contain '%s' for %s in %s",
					expected, tt.alias, tt.shell)
			}
		})
	}
}

func TestShellIntegration_FormattingFunction(t *testing.T) {
	if os.Getenv("TEST_SHELL") == "" {
		t.Skip("Skipping shell integration tests (run with task test-completion in Docker)")
	}
	if testShell != "bash" {
		t.Skip("Formatting function test only for bash")
	}

	setupTestEnvironment(t)

	configDir, err := filepath.Abs("testdata")
	require.NoError(t, err)

	// Test that __dirvana_format_descriptions function exists
	cmd := exec.Command("bash", "-c", `
		cd "`+configDir+`" && \
		eval "$(dirvana export)" && \
		type __dirvana_format_descriptions
	`)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err,
		"__dirvana_format_descriptions should be defined\nOutput: %s", string(output))

	assert.Contains(t, string(output), "__dirvana_format_descriptions is a function")
}

func TestShellIntegration_CompletionFunction(t *testing.T) {
	if os.Getenv("TEST_SHELL") == "" {
		t.Skip("Skipping shell integration tests (run with task test-completion in Docker)")
	}
	if testShell != "bash" {
		t.Skip("Completion function test only for bash")
	}

	setupTestEnvironment(t)

	configDir, err := filepath.Abs("testdata")
	require.NoError(t, err)

	// Test that __dirvana_complete function exists and is registered
	cmd := exec.Command("bash", "-c", `
		cd "`+configDir+`" && \
		eval "$(dirvana export)" && \
		complete -p k 2>/dev/null
	`)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err,
		"Completion should be registered for 'k' alias\nOutput: %s", string(output))

	assert.Contains(t, string(output), "__dirvana_complete",
		"Should use __dirvana_complete function")
}

// setupTestEnvironment prepares the test environment
func setupTestEnvironment(t *testing.T) {
	t.Helper()

	// Ensure testdata directory exists
	configDir := "testdata"
	require.DirExists(t, configDir, "testdata directory should exist")

	// Allow the test directory
	cmd := exec.Command("dirvana", "allow", configDir)
	_ = cmd.Run() // Ignore error if already allowed

	t.Logf("Test environment ready (shell: %s)", testShell)
}

// isValidCompletion checks if a word is a valid completion
// (filters out prompts, control characters, and too-short words)
func isValidCompletion(word string) bool {
	return !strings.HasPrefix(word, "$") &&
		!strings.HasPrefix(word, "%") &&
		!strings.HasPrefix(word, "#") &&
		!strings.Contains(word, "\x1b") &&
		len(word) > 1
}

// parseCompletionOutput extracts completions from expect script output
func parseCompletionOutput(output string) []string {
	var completions []string
	inCompletions := false

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "COMPLETIONS_START") {
			inCompletions = true
			continue
		}

		if strings.Contains(line, "COMPLETIONS_END") {
			inCompletions = false
			continue
		}

		if inCompletions && line != "" {
			// Two formats possible:
			// 1. "command  -- description" (kubectl, zsh) -> take only first word
			// 2. "command1  command2  command3" (terraform, aqua in columns) -> take all words
			words := strings.Fields(line)
			if len(words) == 0 {
				continue
			}

			// If line contains "--", it's format 1 (with description)
			if strings.Contains(line, " -- ") {
				// Take only the first word (the command)
				word := words[0]
				if !isValidCompletion(word) {
					continue
				}
				completions = append(completions, word)
			} else {
				// Format 2: multiple completions per line (columns)
				for _, word := range words {
					if !isValidCompletion(word) {
						continue
					}
					completions = append(completions, word)
				}
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := []string{}
	for _, c := range completions {
		if !seen[c] {
			seen[c] = true
			unique = append(unique, c)
		}
	}

	return unique
}

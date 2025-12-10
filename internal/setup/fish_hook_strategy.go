package setup

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

// FishHookStrategy implements hook installation for Fish shell
// Fish requires special handling because the hook must be inside the `if status is-interactive` block
type FishHookStrategy struct {
	hookPath string
	rcFile   string
	message  string
}

// NewFishHookStrategy creates a new Fish hook strategy
func NewFishHookStrategy() (*FishHookStrategy, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-fish.sh")

	rcFile, err := GetRCFilePath(cli.ShellFish)
	if err != nil {
		return nil, err
	}

	return &FishHookStrategy{
		hookPath: hookPath,
		rcFile:   rcFile,
	}, nil
}

// Install installs the hook for Fish shell
func (s *FishHookStrategy) Install() error {
	// Step 1: Create hook file
	hookCode := cli.GenerateHookCode(cli.ShellFish)

	if err := os.MkdirAll(filepath.Dir(s.hookPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := atomicWrite(s.hookPath, []byte(hookCode)); err != nil {
		return fmt.Errorf("failed to create hook file: %w", err)
	}

	// Step 2: Add source line to config.fish inside is-interactive block
	sourceLine := fmt.Sprintf("    test -f %s; and source %s", s.hookPath, s.hookPath)

	// Check if RC file exists
	data, err := os.ReadFile(s.rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create config.fish with proper structure
			content := fmt.Sprintf(`if status is-interactive
    # Commands to run in interactive sessions can go here
    # Dirvana
%s
end
`, sourceLine)
			if err := os.MkdirAll(filepath.Dir(s.rcFile), 0755); err != nil {
				return fmt.Errorf("failed to create fish config directory: %w", err)
			}
			if err := atomicWrite(s.rcFile, []byte(content)); err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}
			s.message = fmt.Sprintf("✓ Hook created at %s\n✓ Created %s with Dirvana hook", s.hookPath, s.rcFile)
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	content := string(data)

	// Check if already present - look for both the source line and the hook path reference
	if strings.Contains(content, s.hookPath) {
		s.message = fmt.Sprintf("✓ Hook file updated at %s\n✓ Config file already configured", s.hookPath)
		return nil
	}

	// Insert source line inside is-interactive block
	newContent, err := s.insertIntoInteractiveBlock(content, sourceLine)
	if err != nil {
		return fmt.Errorf("failed to insert hook: %w", err)
	}

	if err := atomicWrite(s.rcFile, []byte(newContent)); err != nil {
		return fmt.Errorf("failed to update config file: %w", err)
	}

	s.message = fmt.Sprintf("✓ Hook created at %s\n✓ Added to %s inside is-interactive block", s.hookPath, s.rcFile)
	return nil
}

// insertIntoInteractiveBlock inserts the source line inside the `if status is-interactive` block
func (s *FishHookStrategy) insertIntoInteractiveBlock(content, sourceLine string) (string, error) {
	var result bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader(content))

	foundInteractiveBlock := false
	insertedHook := false
	inInteractiveBlock := false
	blockIndent := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect start of is-interactive block
		if strings.HasPrefix(trimmed, "if status is-interactive") || strings.HasPrefix(trimmed, "if status --is-interactive") {
			foundInteractiveBlock = true
			inInteractiveBlock = true
			result.WriteString(line + "\n")

			// Detect indentation of next line (usually 4 spaces)
			blockIndent = 4
			continue
		}

		// Detect end of is-interactive block
		if inInteractiveBlock && trimmed == "end" {
			// Insert hook before the 'end'
			if !insertedHook {
				result.WriteString(strings.Repeat(" ", blockIndent) + "# Dirvana\n")
				result.WriteString(sourceLine + "\n")
				insertedHook = true
			}
			inInteractiveBlock = false
			result.WriteString(line + "\n")
			continue
		}

		result.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan file: %w", err)
	}

	// If no is-interactive block found, create one at the end
	if !foundInteractiveBlock {
		if !strings.HasSuffix(result.String(), "\n\n") {
			if !strings.HasSuffix(result.String(), "\n") {
				result.WriteString("\n")
			}
			result.WriteString("\n")
		}
		result.WriteString("if status is-interactive\n")
		result.WriteString("    # Commands to run in interactive sessions can go here\n")
		result.WriteString("    # Dirvana\n")
		result.WriteString(sourceLine + "\n")
		result.WriteString("end\n")
		insertedHook = true
	}

	if !insertedHook {
		return "", fmt.Errorf("failed to find appropriate location to insert hook")
	}

	return result.String(), nil
}

// Uninstall removes the hook
func (s *FishHookStrategy) Uninstall() error {
	// Step 1: Remove hook file
	if err := os.Remove(s.hookPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove hook file: %w", err)
	}

	// Step 2: Remove lines from config file
	data, err := os.ReadFile(s.rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.message = "✓ Nothing to uninstall"
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	skipNext := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip Dirvana comment
		if trimmed == "# Dirvana" {
			skipNext = true
			continue
		}

		// Skip source line
		if skipNext && strings.Contains(line, s.hookPath) {
			skipNext = false
			continue
		}

		skipNext = false
		newLines = append(newLines, line)
	}

	newContent := strings.Join(newLines, "\n")
	if err := atomicWrite(s.rcFile, []byte(newContent)); err != nil {
		return fmt.Errorf("failed to update config file: %w", err)
	}

	s.message = fmt.Sprintf("✓ Removed hook file: %s\n✓ Removed line from %s", s.hookPath, s.rcFile)
	return nil
}

// IsInstalled checks if the hook is installed
func (s *FishHookStrategy) IsInstalled() bool {
	// Check if hook file exists
	if _, err := os.Stat(s.hookPath); os.IsNotExist(err) {
		return false
	}

	// Check if config file references it
	data, err := os.ReadFile(s.rcFile)
	if err != nil {
		return false
	}

	return strings.Contains(string(data), s.hookPath)
}

// NeedsUpdate checks if the hook needs to be updated
func (s *FishHookStrategy) NeedsUpdate() bool {
	// Check if hook file content matches current version
	currentHook, err := os.ReadFile(s.hookPath)
	if err != nil {
		return true
	}

	expectedHook := cli.GenerateHookCode(cli.ShellFish)
	return string(currentHook) != expectedHook
}

// GetMessage returns a user-friendly message
func (s *FishHookStrategy) GetMessage() string {
	if s.message == "" {
		return "✓ Dirvana hook is up to date"
	}
	return s.message
}

// GetRCFile returns the config file path
func (s *FishHookStrategy) GetRCFile() string {
	return s.rcFile
}

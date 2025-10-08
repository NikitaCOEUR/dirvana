package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

const (
	// DirvanaComment is the comment added to RC files
	DirvanaComment = "# Dirvana"
)

// ExternalHookStrategy implements hook installation using an external hook file
type ExternalHookStrategy struct {
	shell    string
	hookPath string
	rcFile   string
	message  string
}

// NewExternalHookStrategy creates a new external hook strategy
func NewExternalHookStrategy(shell string) (*ExternalHookStrategy, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, fmt.Sprintf("hook-%s.sh", shell))

	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return nil, err
	}

	return &ExternalHookStrategy{
		shell:    shell,
		hookPath: hookPath,
		rcFile:   rcFile,
	}, nil
}

// Install installs the hook using external file strategy
func (s *ExternalHookStrategy) Install() error {
	// Step 1: Create hook file with all logic
	hookCode := cli.GenerateHookCode(s.shell)

	if err := os.MkdirAll(filepath.Dir(s.hookPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := atomicWrite(s.hookPath, []byte(hookCode)); err != nil {
		return fmt.Errorf("failed to create hook file: %w", err)
	}

	// Step 2: Add single source line to RC file (if not present)
	sourceLine := fmt.Sprintf("[ -f %s ] && source %s", s.hookPath, s.hookPath)

	data, err := os.ReadFile(s.rcFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read RC file: %w", err)
	}

	content := string(data)

	// Check if already present by looking for the source line
	if strings.Contains(content, sourceLine) {
		s.message = fmt.Sprintf("✓ Hook file updated at %s\n✓ RC file already configured", s.hookPath)
		return nil
	}

	// Append single line with comment
	newContent := content
	if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += fmt.Sprintf("\n%s\n%s\n", DirvanaComment, sourceLine)

	if err := atomicWrite(s.rcFile, []byte(newContent)); err != nil {
		return fmt.Errorf("failed to update RC file: %w", err)
	}

	warning := checkDirenvConflict(s.rcFile)
	s.message = fmt.Sprintf("✓ Hook created at %s\n✓ Added single line to %s%s",
		s.hookPath, s.rcFile, warning)

	return nil
}

// Uninstall removes the hook
func (s *ExternalHookStrategy) Uninstall() error {
	// Step 1: Remove hook file
	if err := os.Remove(s.hookPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove hook file: %w", err)
	}

	// Step 2: Remove lines from RC file
	data, err := os.ReadFile(s.rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.message = "✓ Nothing to uninstall"
			return nil
		}
		return fmt.Errorf("failed to read RC file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	skipNext := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment line
		if trimmed == DirvanaComment {
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
		return fmt.Errorf("failed to update RC file: %w", err)
	}

	s.message = fmt.Sprintf("✓ Removed hook file: %s\n✓ Removed line from %s", s.hookPath, s.rcFile)
	return nil
}

// IsInstalled checks if the hook is installed
func (s *ExternalHookStrategy) IsInstalled() bool {
	// Check if hook file exists
	if _, err := os.Stat(s.hookPath); os.IsNotExist(err) {
		return false
	}

	// Check if RC file references it
	data, err := os.ReadFile(s.rcFile)
	if err != nil {
		return false
	}

	return strings.Contains(string(data), s.hookPath)
}

// NeedsUpdate checks if the hook needs to be updated
func (s *ExternalHookStrategy) NeedsUpdate() bool {
	// Check if hook file content matches current version
	currentHook, err := os.ReadFile(s.hookPath)
	if err != nil {
		return true
	}

	expectedHook := cli.GenerateHookCode(s.shell)
	return string(currentHook) != expectedHook
}

// GetMessage returns a user-friendly message
func (s *ExternalHookStrategy) GetMessage() string {
	if s.message == "" {
		return "✓ Dirvana hook is up to date"
	}
	return s.message
}

// GetRCFile returns the RC file path
func (s *ExternalHookStrategy) GetRCFile() string {
	return s.rcFile
}

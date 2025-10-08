// Package setup provides shell hook installation strategies for dirvana.
package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

// DropInStrategy implements hook installation using shell drop-in directories
// This is the cleanest approach when supported (e.g., .bashrc.d on Ubuntu/Debian)
type DropInStrategy struct {
	shell      string
	dropInDir  string
	dropInFile string
	rcFile     string
	message    string
}

// NewDropInStrategy creates a new drop-in strategy
func NewDropInStrategy(shell string) (*DropInStrategy, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dropInDir := filepath.Join(home, fmt.Sprintf(".%src.d", shell))
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return nil, err
	}

	return &DropInStrategy{
		shell:      shell,
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}, nil
}

// IsSupported checks if the shell RC file already sources drop-in directory
func (s *DropInStrategy) IsSupported() bool {
	data, err := os.ReadFile(s.rcFile)
	if err != nil {
		return false
	}

	content := string(data)
	dropInPattern := fmt.Sprintf(".%src.d", s.shell)

	// Check if RC file sources the drop-in directory
	return strings.Contains(content, dropInPattern)
}

// Install installs the hook using drop-in directory
func (s *DropInStrategy) Install() error {
	// Create drop-in directory if needed
	if err := os.MkdirAll(s.dropInDir, 0755); err != nil {
		return fmt.Errorf("failed to create drop-in directory: %w", err)
	}

	hookCode := cli.GenerateHookCode(s.shell)
	if err := atomicWrite(s.dropInFile, []byte(hookCode)); err != nil {
		return fmt.Errorf("failed to create drop-in file: %w", err)
	}

	warning := checkDirenvConflict(s.rcFile)
	s.message = fmt.Sprintf("✓ Hook installed to %s\n✓ No modification to %s needed!%s",
		s.dropInFile, s.rcFile, warning)

	return nil
}

// Uninstall removes the drop-in file
func (s *DropInStrategy) Uninstall() error {
	if err := os.Remove(s.dropInFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove drop-in file: %w", err)
	}

	s.message = fmt.Sprintf("✓ Removed %s", s.dropInFile)
	return nil
}

// IsInstalled checks if the drop-in file exists
func (s *DropInStrategy) IsInstalled() bool {
	_, err := os.Stat(s.dropInFile)
	return err == nil
}

// NeedsUpdate checks if the hook needs to be updated
func (s *DropInStrategy) NeedsUpdate() bool {
	// Check if hook file content matches current version
	currentHook, err := os.ReadFile(s.dropInFile)
	if err != nil {
		return true
	}

	expectedHook := cli.GenerateHookCode(s.shell)
	return string(currentHook) != expectedHook
}

// GetMessage returns a user-friendly message
func (s *DropInStrategy) GetMessage() string {
	if s.message == "" {
		return "✓ Dirvana hook is up to date"
	}
	return s.message
}

// GetRCFile returns the RC file path
func (s *DropInStrategy) GetRCFile() string {
	return s.rcFile
}

package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

// InstallStrategy defines the interface for different hook installation strategies
type InstallStrategy interface {
	// Install installs the hook using the strategy
	Install() error
	// Uninstall removes the hook
	Uninstall() error
	// IsInstalled checks if the hook is currently installed
	IsInstalled() bool
	// NeedsUpdate checks if the hook needs to be updated
	NeedsUpdate() bool
	// GetMessage returns a user-friendly message about the installation
	GetMessage() string
	// GetRCFile returns the RC file path (if applicable)
	GetRCFile() string
}

// SelectInstallStrategy selects the best installation strategy for the given shell
func SelectInstallStrategy(shell string) (InstallStrategy, error) {
	// Fish requires special handling due to is-interactive block
	if shell == cli.ShellFish {
		return NewFishHookStrategy()
	}

	// Try drop-in strategy first (cleanest approach)
	dropIn, err := NewDropInStrategy(shell)
	if err == nil && dropIn.IsSupported() {
		return dropIn, nil
	}

	// Fallback to external hook strategy
	return NewExternalHookStrategy(shell)
}

// HasLegacyInstall checks if there's a legacy installation (old inline hook)
func HasLegacyInstall(shell string) bool {
	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return false
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		return false
	}

	return containsMarkers(string(data), HookMarkerStart, HookMarkerEnd)
}

// MigrateLegacyInstall migrates from legacy inline hook to new strategy
func MigrateLegacyInstall(shell string) error {
	if !HasLegacyInstall(shell) {
		return nil
	}

	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return err
	}

	// Remove legacy hook
	if err := removeLegacyHook(rcFile); err != nil {
		return fmt.Errorf("failed to remove legacy hook: %w", err)
	}

	// Install using new strategy
	strategy, err := SelectInstallStrategy(shell)
	if err != nil {
		return fmt.Errorf("failed to select strategy: %w", err)
	}

	if err := strategy.Install(); err != nil {
		return fmt.Errorf("failed to install with new strategy: %w", err)
	}

	return nil
}

// removeLegacyHook removes the old inline hook from RC file
func removeLegacyHook(rcFile string) error {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		return err
	}

	content := string(data)
	newContent := removeMarkedSection(content, HookMarkerStart, HookMarkerEnd)

	return atomicWrite(rcFile, []byte(newContent))
}

// atomicWrite writes data to a file atomically using a temporary file
func atomicWrite(filename string, data []byte) error {
	const perm = 0644
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, ".dirvana-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	// Clean up temp file if something goes wrong
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Success - don't clean up temp file
	tmpFile = nil
	return nil
}

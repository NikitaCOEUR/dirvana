package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

const (
	// HookMarkerStart is the starting marker for Dirvana hook in RC files
	HookMarkerStart = "# Dirvana shell hook - START"
	// HookMarkerEnd is the ending marker for Dirvana hook in RC files
	HookMarkerEnd = "# Dirvana shell hook - END"
)

// Result represents the result of a setup operation
type Result struct {
	RCFile  string
	Updated bool
	Message string
}

// GetRCFilePath returns the RC file path for the given shell
func GetRCFilePath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch shell {
	case cli.ShellBash:
		return filepath.Join(home, ".bashrc"), nil
	case cli.ShellZsh:
		return filepath.Join(home, ".zshrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (use bash or zsh)", shell)
	}
}

// checkDirenvConflict checks if direnv is installed and warns the user
func checkDirenvConflict(rcFile string) string {
	// Read RC file to check for direnv
	data, err := os.ReadFile(rcFile)
	if err != nil {
		return ""
	}

	content := string(data)
	if strings.Contains(content, "direnv") {
		return "\n⚠️  Warning: direnv detected in your shell configuration.\n" +
			"   Dirvana and direnv may conflict. Consider using only one of them.\n" +
			"   If you experience issues, remove direnv hooks from your shell config."
	}

	return ""
}

// InstallHook installs or updates the Dirvana hook using the best strategy
func InstallHook(shell string) (*Result, error) {
	// Check for legacy installation
	if HasLegacyInstall(shell) {
		// Migrate from legacy to new strategy
		if err := MigrateLegacyInstall(shell); err != nil {
			return nil, fmt.Errorf("failed to migrate legacy installation: %w", err)
		}

		strategy, err := SelectInstallStrategy(shell)
		if err != nil {
			return nil, err
		}

		return &Result{
			RCFile:  strategy.GetRCFile(),
			Updated: true,
			Message: "⚠️  Migrated from legacy installation\n" + strategy.GetMessage() + "\n✓ Shell completion is up to date",
		}, nil
	}

	// Use new strategy-based installation
	strategy, err := SelectInstallStrategy(shell)
	if err != nil {
		return nil, err
	}

	// Check if already installed and up to date
	if strategy.IsInstalled() && !strategy.NeedsUpdate() {
		return &Result{
			RCFile:  strategy.GetRCFile(),
			Updated: false,
			Message: strategy.GetMessage() + "\n✓ Shell completion is up to date",
		}, nil
	}

	// Install or update
	if err := strategy.Install(); err != nil {
		return nil, fmt.Errorf("failed to install hook: %w", err)
	}

	return &Result{
		RCFile:  strategy.GetRCFile(),
		Updated: true,
		Message: strategy.GetMessage() + "\n✓ Shell completion is up to date",
	}, nil
}

// IsHookInstalled checks if the Dirvana hook is installed (legacy or new strategy)
func IsHookInstalled(shell string) (bool, error) {
	// Check for legacy installation
	if HasLegacyInstall(shell) {
		return true, nil
	}

	// Check with new strategy
	strategy, err := SelectInstallStrategy(shell)
	if err != nil {
		return false, err
	}

	return strategy.IsInstalled(), nil
}

// UninstallHook removes the Dirvana hook (handles both legacy and new strategies)
func UninstallHook(shell string) (*Result, error) {
	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return nil, err
	}

	// Check if legacy install exists
	legacyExists := HasLegacyInstall(shell)

	// Check if new strategy is installed
	strategy, err := SelectInstallStrategy(shell)
	if err != nil {
		return nil, err
	}
	newStrategyInstalled := strategy.IsInstalled()

	// If nothing is installed, return early
	if !legacyExists && !newStrategyInstalled {
		return &Result{
			RCFile:  rcFile,
			Updated: false,
			Message: "✓ Dirvana is not installed",
		}, nil
	}

	// Remove legacy if it exists
	if legacyExists {
		if err := uninstallLegacyHook(shell); err != nil {
			return nil, err
		}
	}

	// Remove new strategy if it's installed
	if newStrategyInstalled {
		if err := strategy.Uninstall(); err != nil {
			return nil, fmt.Errorf("failed to uninstall: %w", err)
		}
	}

	// Build message based on what was removed
	var message string
	if legacyExists && newStrategyInstalled {
		message = "✓ Removed legacy hook and new hook"
	} else if legacyExists {
		message = fmt.Sprintf("✓ Removed legacy hook from %s", rcFile)
	} else {
		message = strategy.GetMessage()
	}

	return &Result{
		RCFile:  rcFile,
		Updated: true,
		Message: message,
	}, nil
}

// uninstallLegacyHook removes legacy hook from RC file
func uninstallLegacyHook(shell string) error {
	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return err
	}

	return removeLegacyHook(rcFile)
}

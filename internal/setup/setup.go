package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/shell"
)

// Result represents the result of a setup operation
type Result struct {
	RCFile  string
	Updated bool
	Message string
}

// GetRCFilePath returns the RC file path for the given shell
func GetRCFilePath(shellName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch shellName {
	case shell.Bash:
		return filepath.Join(home, ".bashrc"), nil
	case shell.Zsh:
		return filepath.Join(home, ".zshrc"), nil
	case shell.Fish:
		return filepath.Join(home, ".config/fish/config.fish"), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (use bash, zsh, or fish)", shellName)
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

// IsHookInstalled checks if the Dirvana hook is installed
func IsHookInstalled(shell string) (bool, error) {
	strategy, err := SelectInstallStrategy(shell)
	if err != nil {
		return false, err
	}

	return strategy.IsInstalled(), nil
}

// UninstallHook removes the Dirvana hook
func UninstallHook(shell string) (*Result, error) {
	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return nil, err
	}

	strategy, err := SelectInstallStrategy(shell)
	if err != nil {
		return nil, err
	}

	if !strategy.IsInstalled() {
		return &Result{
			RCFile:  rcFile,
			Updated: false,
			Message: "✓ Dirvana is not installed",
		}, nil
	}

	if err := strategy.Uninstall(); err != nil {
		return nil, fmt.Errorf("failed to uninstall: %w", err)
	}

	return &Result{
		RCFile:  rcFile,
		Updated: true,
		Message: strategy.GetMessage(),
	}, nil
}

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// HookMarkerStart is the starting marker for Dirvana hook in RC files
	HookMarkerStart = "# Dirvana shell hook - START"
	// HookMarkerEnd is the ending marker for Dirvana hook in RC files
	HookMarkerEnd = "# Dirvana shell hook - END"
)

// SetupResult represents the result of a setup operation
type SetupResult struct {
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
	case ShellBash:
		return filepath.Join(home, ".bashrc"), nil
	case ShellZsh:
		return filepath.Join(home, ".zshrc"), nil
	case ShellPowerShell:
		// Windows PowerShell profile
		return filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"), nil
	case ShellPwsh:
		// PowerShell Core profile
		return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (use bash, zsh, powershell, or pwsh)", shell)
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

// InstallHook installs or updates the Dirvana hook in the RC file
func InstallHook(shell string) (*SetupResult, error) {
	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return nil, err
	}

	hookCode := GenerateHookCode(shell)
	hookBlock := fmt.Sprintf("%s\n%s\n%s", HookMarkerStart, hookCode, HookMarkerEnd)

	// Read existing rc file
	data, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read %s: %w", rcFile, err)
	}

	content := string(data)

	// Check if hook is already installed
	startIdx := strings.Index(content, HookMarkerStart)
	endIdx := strings.Index(content, HookMarkerEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Hook exists - extract current hook
		currentHook := content[startIdx : endIdx+len(HookMarkerEnd)]

		// Compare with new hook
		if currentHook == hookBlock {
			warning := checkDirenvConflict(rcFile)
			message := fmt.Sprintf("✓ Dirvana hook is up to date in %s%s\n✓ Shell completion is up to date", rcFile, warning)

			return &SetupResult{
				RCFile:  rcFile,
				Updated: false,
				Message: message,
			}, nil
		}

		// Replace old hook with new one
		newContent := content[:startIdx] + hookBlock + content[endIdx+len(HookMarkerEnd):]
		if err := os.WriteFile(rcFile, []byte(newContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to update %s: %w", rcFile, err)
		}

		warning := checkDirenvConflict(rcFile)
		message := fmt.Sprintf("✓ Dirvana hook updated in %s%s\n✓ Shell completion is up to date", rcFile, warning)

		return &SetupResult{
			RCFile:  rcFile,
			Updated: true,
			Message: message,
		}, nil
	}

	// Hook not installed - add it
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", rcFile, err)
	}

	if _, err := f.WriteString("\n" + hookBlock + "\n"); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to write to %s: %w", rcFile, err)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("failed to close %s: %w", rcFile, err)
	}

	warning := checkDirenvConflict(rcFile)
	message := fmt.Sprintf("✓ Dirvana hook installed in %s%s", rcFile, warning)

	return &SetupResult{
		RCFile:  rcFile,
		Updated: true,
		Message: message,
	}, nil
}

// IsHookInstalled checks if the Dirvana hook is already installed in the RC file
func IsHookInstalled(shell string) (bool, error) {
	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return false, err
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read %s: %w", rcFile, err)
	}

	content := string(data)
	return strings.Contains(content, HookMarkerStart) && strings.Contains(content, HookMarkerEnd), nil
}

// UninstallHook removes the Dirvana hook from the RC file
func UninstallHook(shell string) (*SetupResult, error) {
	rcFile, err := GetRCFilePath(shell)
	if err != nil {
		return nil, err
	}

	// Read existing rc file
	data, err := os.ReadFile(rcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &SetupResult{
				RCFile:  rcFile,
				Updated: false,
				Message: fmt.Sprintf("✓ Nothing to uninstall (file doesn't exist): %s", rcFile),
			}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", rcFile, err)
	}

	content := string(data)

	// Check if hook is installed
	startIdx := strings.Index(content, HookMarkerStart)
	endIdx := strings.Index(content, HookMarkerEnd)

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return &SetupResult{
			RCFile:  rcFile,
			Updated: false,
			Message: fmt.Sprintf("✓ Dirvana is not installed in %s", rcFile),
		}, nil
	}

	// Remove hook block (including the newlines around it)
	before := content[:startIdx]
	after := content[endIdx+len(HookMarkerEnd):]

	// Trim extra newlines
	before = strings.TrimRight(before, "\n")
	after = strings.TrimLeft(after, "\n")

	content = before + "\n" + after

	// Write updated content
	if err := os.WriteFile(rcFile, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", rcFile, err)
	}

	return &SetupResult{
		RCFile:  rcFile,
		Updated: true,
		Message: fmt.Sprintf("✓ Dirvana hook removed from %s", rcFile),
	}, nil
}

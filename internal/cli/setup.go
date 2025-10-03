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

	// CompletionMarkerStart is the starting marker for Dirvana completion in RC files
	CompletionMarkerStart = "# Dirvana shell completion - START"
	// CompletionMarkerEnd is the ending marker for Dirvana completion in RC files
	CompletionMarkerEnd = "# Dirvana shell completion - END"

	// CompletionInstalledMsg is the message displayed when completion is installed
	CompletionInstalledMsg = "\n✓ Shell completion installed"
	// CompletionUpdatedMsg is the message displayed when completion is updated
	CompletionUpdatedMsg = "\n✓ Shell completion updated"
)

// SetupResult represents the result of a setup operation
type SetupResult struct {
	RCFile              string
	Updated             bool
	Message             string
	CompletionInstalled bool
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

			// Completion is now handled by dirvana export, no need for static completion
			// completionChanged, wasUpdate, _ := InstallCompletion(shell, rcFile)
			completionChanged, wasUpdate := false, false
			_ = wasUpdate // unused

			message := fmt.Sprintf("✓ Dirvana hook is up to date in %s%s", rcFile, warning)
			if completionChanged {
				if wasUpdate {
					message += CompletionUpdatedMsg
				} else {
					message += CompletionInstalledMsg
				}
			} else {
				message += "\n✓ Shell completion is up to date"
			}

			return &SetupResult{
				RCFile:              rcFile,
				Updated:             false,
				Message:             message,
				CompletionInstalled: completionChanged,
			}, nil
		}

		// Replace old hook with new one
		newContent := content[:startIdx] + hookBlock + content[endIdx+len(HookMarkerEnd):]
		if err := os.WriteFile(rcFile, []byte(newContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to update %s: %w", rcFile, err)
		}

		warning := checkDirenvConflict(rcFile)

		// Completion is now handled by dirvana export, no need for static completion
		// completionChanged, wasUpdate, _ := InstallCompletion(shell, rcFile)
		completionChanged, wasUpdate := false, false
		_ = wasUpdate // unused

		message := fmt.Sprintf("✓ Dirvana hook updated in %s%s", rcFile, warning)
		if completionChanged {
			if wasUpdate {
				message += CompletionUpdatedMsg
			} else {
				message += CompletionInstalledMsg
			}
		} else {
			message += "\n✓ Shell completion is up to date"
		}

		return &SetupResult{
			RCFile:              rcFile,
			Updated:             true,
			Message:             message,
			CompletionInstalled: completionChanged,
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

	// Completion is now handled by dirvana export, no need for static completion
	// completionChanged, wasUpdate, _ := InstallCompletion(shell, rcFile)
	completionChanged, wasUpdate := false, false
	_ = wasUpdate // unused

	message := fmt.Sprintf("✓ Dirvana hook installed in %s%s", rcFile, warning)
	if completionChanged {
		if wasUpdate {
			message += CompletionUpdatedMsg
		} else {
			message += CompletionInstalledMsg
		}
	}

	return &SetupResult{
		RCFile:              rcFile,
		Updated:             true,
		Message:             message,
		CompletionInstalled: completionChanged,
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

// InstallCompletion installs or updates shell completion in the RC file
// Returns (changed, wasUpdate, error) where:
// - changed: true if completion was installed or updated
// - wasUpdate: true if it was an update, false if it was a new installation
func InstallCompletion(shell string, rcFile string) (bool, bool, error) {
	// Get the path to the current binary
	binPath, err := os.Executable()
	if err != nil {
		binPath = "dirvana" // Fallback to PATH
	}

	// Generate completion code based on shell
	var completionCode string
	switch shell {
	case ShellBash:
		completionCode = fmt.Sprintf("command -v %s &> /dev/null && source <(%s completion bash)", binPath, binPath)
	case ShellZsh:
		completionCode = fmt.Sprintf("command -v %s &> /dev/null && source <(%s completion zsh)", binPath, binPath)
	case ShellPowerShell, ShellPwsh:
		completionCode = fmt.Sprintf("if (Get-Command %s -ErrorAction SilentlyContinue) { & %s completion powershell | Out-String | Invoke-Expression }", binPath, binPath)
	default:
		return false, false, fmt.Errorf("unsupported shell for completion: %s", shell)
	}

	completionBlock := fmt.Sprintf("%s\n%s\n%s", CompletionMarkerStart, completionCode, CompletionMarkerEnd)

	// Read existing rc file
	data, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return false, false, fmt.Errorf("failed to read %s: %w", rcFile, err)
	}

	content := string(data)

	// Check if completion is already installed
	startIdx := strings.Index(content, CompletionMarkerStart)
	endIdx := strings.Index(content, CompletionMarkerEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Completion exists - extract current completion
		currentCompletion := content[startIdx : endIdx+len(CompletionMarkerEnd)]

		// Compare with new completion
		if currentCompletion == completionBlock {
			return false, false, nil // Already up to date
		}

		// Replace old completion with new one
		newContent := content[:startIdx] + completionBlock + content[endIdx+len(CompletionMarkerEnd):]
		if err := os.WriteFile(rcFile, []byte(newContent), 0644); err != nil {
			return false, false, fmt.Errorf("failed to update %s: %w", rcFile, err)
		}

		return true, true, nil // Changed, was an update
	}

	// Completion not installed - add it
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, false, fmt.Errorf("failed to open %s: %w", rcFile, err)
	}

	if _, err := f.WriteString("\n" + completionBlock + "\n"); err != nil {
		_ = f.Close()
		return false, false, fmt.Errorf("failed to write to %s: %w", rcFile, err)
	}

	if err := f.Close(); err != nil {
		return false, false, fmt.Errorf("failed to close %s: %w", rcFile, err)
	}

	return true, false, nil // Changed, was a new installation
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

	hookRemoved := false
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Remove hook block (including the newlines around it)
		before := content[:startIdx]
		after := content[endIdx+len(HookMarkerEnd):]

		// Trim extra newlines
		before = strings.TrimRight(before, "\n")
		after = strings.TrimLeft(after, "\n")

		content = before + "\n" + after
		hookRemoved = true
	}

	// Check if completion is installed
	startIdx = strings.Index(content, CompletionMarkerStart)
	endIdx = strings.Index(content, CompletionMarkerEnd)

	completionRemoved := false
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Remove completion block
		before := content[:startIdx]
		after := content[endIdx+len(CompletionMarkerEnd):]

		// Trim extra newlines
		before = strings.TrimRight(before, "\n")
		after = strings.TrimLeft(after, "\n")

		content = before + "\n" + after
		completionRemoved = true
	}

	if !hookRemoved && !completionRemoved {
		return &SetupResult{
			RCFile:  rcFile,
			Updated: false,
			Message: fmt.Sprintf("✓ Dirvana is not installed in %s", rcFile),
		}, nil
	}

	// Write updated content
	if err := os.WriteFile(rcFile, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", rcFile, err)
	}

	message := fmt.Sprintf("✓ Dirvana uninstalled from %s", rcFile)
	if hookRemoved && completionRemoved {
		message += "\n  • Hook removed\n  • Completion removed"
	} else if hookRemoved {
		message += "\n  • Hook removed"
	} else if completionRemoved {
		message += "\n  • Completion removed"
	}

	return &SetupResult{
		RCFile:  rcFile,
		Updated: true,
		Message: message,
	}, nil
}

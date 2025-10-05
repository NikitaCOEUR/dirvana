//go:build linux

package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectShellFromParentProcess_Linux(t *testing.T) {
	// On Linux, this should try to read /proc/$PPID/cmdline
	ppid := os.Getppid()
	
	// Check if /proc/$PPID/cmdline exists
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", ppid)
	if _, err := os.Stat(cmdlinePath); os.IsNotExist(err) {
		t.Skip("Parent process cmdline not available")
	}
	
	// Call the function
	result := detectShellFromParentProcess()
	
	// On Linux, if the test is running under bash or zsh, we should detect it
	// Otherwise, it might return empty string (if parent is not a shell)
	assert.Contains(t, []string{"", ShellBash, ShellZsh}, result,
		"Should return valid shell or empty string")
	
	// If we got a result, verify it matches the actual parent process
	if result != "" {
		cmdlineData, err := os.ReadFile(cmdlinePath)
		assert.NoError(t, err)
		
		cmdline := string(cmdlineData)
		assert.Contains(t, cmdline, result,
			"Detected shell should match parent process cmdline")
	}
}

func TestDetectShell_UsesParentProcess_Linux(t *testing.T) {
	// Clear environment variables to force parent process detection
	originalDirvanaShell := os.Getenv("DIRVANA_SHELL")
	originalShell := os.Getenv("SHELL")
	
	_ = os.Unsetenv("DIRVANA_SHELL")
	_ = os.Unsetenv("SHELL")
	
	// Restore after test
	defer func() {
		if originalDirvanaShell != "" {
			_ = os.Setenv("DIRVANA_SHELL", originalDirvanaShell)
		}
		if originalShell != "" {
			_ = os.Setenv("SHELL", originalShell)
		}
	}()
	
	// This should trigger the parent process detection path
	result := DetectShell("auto")
	
	// Should return a valid shell (either detected or fallback to bash)
	assert.Contains(t, []string{ShellBash, ShellZsh}, result)
	
	// If parent process detection worked, the result should match
	// what detectShellFromParentProcess returns
	parentShell := detectShellFromParentProcess()
	if parentShell != "" {
		assert.Equal(t, parentShell, result,
			"When parent process detection works, DetectShell should use it")
	}
}

func TestDetectShellFromParentProcess_Coverage(t *testing.T) {
	// This test explicitly exercises detectShellFromParentProcess
	// to ensure the parent detection path is covered
	result := detectShellFromParentProcess()
	
	// On Linux with /proc available, this might return a shell
	// Otherwise returns empty string
	assert.Contains(t, []string{"", ShellBash, ShellZsh}, result)
	
	// If we got a shell back, verify it's from our parent
	if result != "" {
		ppid := os.Getppid()
		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", ppid)
		if data, err := os.ReadFile(cmdlinePath); err == nil {
			// The detected shell should appear in the parent's cmdline
			cmdline := string(data)
			assert.True(t, 
				contains(cmdline, result),
				"Detected shell %s should be in parent cmdline: %s", result, cmdline)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

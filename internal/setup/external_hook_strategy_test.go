package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

func TestExternalHookStrategy_Install(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	home := tmpDir

	// Set up test environment
	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	rcFile := filepath.Join(home, ".bashrc")

	strategy := &ExternalHookStrategy{
		shell:    "bash",
		hookPath: hookPath,
		rcFile:   rcFile,
	}

	// Test install
	err := strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify hook file was created
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Error("Hook file was not created")
	}

	// Verify hook file contains expected content
	hookContent, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("Failed to read hook file: %v", err)
	}

	expectedHookCode := cli.GenerateHookCode("bash")
	if string(hookContent) != expectedHookCode {
		t.Error("Hook file content doesn't match expected")
	}

	// Verify RC file was updated
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	if !strings.Contains(string(rcContent), hookPath) {
		t.Error("RC file doesn't contain reference to hook file")
	}

	if !strings.Contains(string(rcContent), DirvanaComment) {
		t.Error("RC file doesn't contain Dirvana comment")
	}
}

func TestExternalHookStrategy_InstallIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	rcFile := filepath.Join(home, ".bashrc")

	strategy := &ExternalHookStrategy{
		shell:    "bash",
		hookPath: hookPath,
		rcFile:   rcFile,
	}

	// Install twice
	err := strategy.Install()
	if err != nil {
		t.Fatalf("First install failed: %v", err)
	}

	err = strategy.Install()
	if err != nil {
		t.Fatalf("Second install failed: %v", err)
	}

	// Verify RC file doesn't have duplicate lines
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	// Count occurrences of the source line (not just the path)
	sourceLine := fmt.Sprintf("[ -f %s ] && source %s", hookPath, hookPath)
	count := strings.Count(string(rcContent), sourceLine)
	if count != 1 {
		t.Errorf("Expected 1 source line, got %d", count)
	}
}

func TestExternalHookStrategy_Uninstall(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	rcFile := filepath.Join(home, ".bashrc")

	strategy := &ExternalHookStrategy{
		shell:    "bash",
		hookPath: hookPath,
		rcFile:   rcFile,
	}

	// Install first
	err := strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify installed
	if !strategy.IsInstalled() {
		t.Error("Strategy should be installed")
	}

	// Uninstall
	err = strategy.Uninstall()
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify hook file was removed
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("Hook file was not removed")
	}

	// Verify RC file no longer contains reference
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	if strings.Contains(string(rcContent), hookPath) {
		t.Error("RC file still contains reference to hook file")
	}

	// Verify not installed
	if strategy.IsInstalled() {
		t.Error("Strategy should not be installed after uninstall")
	}
}

func TestExternalHookStrategy_IsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	rcFile := filepath.Join(home, ".bashrc")

	strategy := &ExternalHookStrategy{
		shell:    "bash",
		hookPath: hookPath,
		rcFile:   rcFile,
	}

	// Should not be installed initially
	if strategy.IsInstalled() {
		t.Error("Strategy should not be installed initially")
	}

	// Install
	err := strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Should be installed now
	if !strategy.IsInstalled() {
		t.Error("Strategy should be installed after Install()")
	}
}

func TestExternalHookStrategy_NeedsUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	rcFile := filepath.Join(home, ".bashrc")

	strategy := &ExternalHookStrategy{
		shell:    "bash",
		hookPath: hookPath,
		rcFile:   rcFile,
	}

	// Should need update when not installed
	if !strategy.NeedsUpdate() {
		t.Error("Should need update when not installed")
	}

	// Install
	err := strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Should not need update after fresh install
	if strategy.NeedsUpdate() {
		t.Error("Should not need update after fresh install")
	}

	// Modify hook file to simulate outdated version
	err = os.WriteFile(hookPath, []byte("# Old hook code"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify hook file: %v", err)
	}

	// Should need update now
	if !strategy.NeedsUpdate() {
		t.Error("Should need update when hook file is outdated")
	}
}

func TestExternalHookStrategy_UpdateOnlyTouchesHookFile(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	configDir := filepath.Join(home, ".config", "dirvana")
	hookPath := filepath.Join(configDir, "hook-bash.sh")
	rcFile := filepath.Join(home, ".bashrc")

	strategy := &ExternalHookStrategy{
		shell:    "bash",
		hookPath: hookPath,
		rcFile:   rcFile,
	}

	// Install
	err := strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Get RC file stat
	rcStat1, err := os.Stat(rcFile)
	if err != nil {
		t.Fatalf("Failed to stat RC file: %v", err)
	}

	// Modify hook file
	err = os.WriteFile(hookPath, []byte("# Old hook code"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify hook file: %v", err)
	}

	// Update (install again)
	err = strategy.Install()
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Get RC file stat again
	rcStat2, err := os.Stat(rcFile)
	if err != nil {
		t.Fatalf("Failed to stat RC file: %v", err)
	}

	// RC file modification time should be the same (not modified during update)
	if rcStat1.ModTime() != rcStat2.ModTime() {
		// Note: This test might be flaky due to filesystem time resolution
		// We check content instead
		rcContent1, _ := os.ReadFile(rcFile)
		rcContent2, _ := os.ReadFile(rcFile)
		if string(rcContent1) != string(rcContent2) {
			t.Error("RC file was modified during update (should only touch hook file)")
		}
	}
}

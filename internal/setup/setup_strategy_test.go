package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectInstallStrategy_PrefersDropIn(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	// Create RC file with drop-in support
	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent := "if [ -d ~/.bashrc.d ]; then\n  for rc in ~/.bashrc.d/*.sh; do\n    source $rc\n  done\nfi"
	err := os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	strategy, err := SelectInstallStrategy("bash")
	if err != nil {
		t.Fatalf("SelectInstallStrategy failed: %v", err)
	}

	// Should return DropInStrategy
	if _, ok := strategy.(*DropInStrategy); !ok {
		t.Errorf("Expected DropInStrategy, got %T", strategy)
	}
}

func TestSelectInstallStrategy_FallbackToExternal(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	// Create RC file without drop-in support
	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent := "# Regular .bashrc"
	err := os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	strategy, err := SelectInstallStrategy("bash")
	if err != nil {
		t.Fatalf("SelectInstallStrategy failed: %v", err)
	}

	// Should return ExternalHookStrategy
	if _, ok := strategy.(*ExternalHookStrategy); !ok {
		t.Errorf("Expected ExternalHookStrategy, got %T", strategy)
	}
}

func TestSelectInstallStrategy_Fish(t *testing.T) {
	strategy, err := SelectInstallStrategy("fish")
	if err != nil {
		t.Fatalf("SelectInstallStrategy failed: %v", err)
	}

	// Should return FishHookStrategy
	if _, ok := strategy.(*FishHookStrategy); !ok {
		t.Errorf("Expected FishHookStrategy, got %T", strategy)
	}
}

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Test writing new file
	content := []byte("test content")
	err := atomicWrite(testFile, content)
	if err != nil {
		t.Fatalf("atomicWrite failed: %v", err)
	}

	// Verify file was created with correct content
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("File content = %q, want %q", string(readContent), string(content))
	}

	// Test overwriting existing file
	newContent := []byte("new test content")
	err = atomicWrite(testFile, newContent)
	if err != nil {
		t.Fatalf("atomicWrite failed on overwrite: %v", err)
	}

	// Verify file was updated
	readContent, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(newContent) {
		t.Errorf("File content = %q, want %q", string(readContent), string(newContent))
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".dirvana-tmp-") {
			t.Errorf("Temporary file left behind: %s", entry.Name())
		}
	}
}

func TestAtomicWrite_InvalidDirectory(t *testing.T) {
	// Try to write to a non-existent directory
	invalidPath := "/nonexistent/path/that/does/not/exist/file.txt"
	err := atomicWrite(invalidPath, []byte("test"))
	if err == nil {
		t.Error("atomicWrite should fail with non-existent directory")
	}
}

func TestAtomicWrite_PermissionVerification(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "perm_test.txt")

	content := []byte("permission test")
	err := atomicWrite(testFile, content)
	if err != nil {
		t.Fatalf("atomicWrite failed: %v", err)
	}

	// Check file has correct permissions (0644)
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	expectedPerm := os.FileMode(0644)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("File permissions = %v, want %v", info.Mode().Perm(), expectedPerm)
	}
}

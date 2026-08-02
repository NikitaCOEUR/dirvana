package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectInstallStrategy_PrefersDropIn(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create RC file with drop-in support
	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent := "if [ -d ~/.bashrc.d ]; then\n  for rc in ~/.bashrc.d/*.sh; do\n    source $rc\n  done\nfi"
	err := os.WriteFile(rcFile, []byte(rcContent), 0o644)
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
	t.Setenv("HOME", tmpDir)

	// Create RC file without drop-in support
	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent := "# Regular .bashrc"
	err := os.WriteFile(rcFile, []byte(rcContent), 0o644)
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

func TestAtomicWrite_RCFilePermissions(t *testing.T) {
	// setup's atomicWrite wraps fsutil.AtomicWrite with the 0644 mode
	// expected for shell RC and hook files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "perm_test.txt")

	if err := atomicWrite(testFile, []byte("permission test")); err != nil {
		t.Fatalf("atomicWrite failed: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	expectedPerm := os.FileMode(0o644)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("File permissions = %v, want %v", info.Mode().Perm(), expectedPerm)
	}
}

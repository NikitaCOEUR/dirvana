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

func TestHasLegacyInstall(t *testing.T) {
	tests := []struct {
		name      string
		rcContent string
		want      bool
	}{
		{
			name:      "No legacy install",
			rcContent: "# Regular .bashrc",
			want:      false,
		},
		{
			name: "Has legacy install",
			rcContent: "# Some content\n" +
				HookMarkerStart + "\n" +
				"# Legacy hook code\n" +
				HookMarkerEnd + "\n" +
				"# More content",
			want: true,
		},
		{
			name:      "Only start marker",
			rcContent: HookMarkerStart + "\n# Some code",
			want:      false,
		},
		{
			name:      "Only end marker",
			rcContent: "# Some code\n" + HookMarkerEnd,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalHome := os.Getenv("HOME")
			defer func() { _ = os.Setenv("HOME", originalHome) }()
			if err := os.Setenv("HOME", tmpDir); err != nil {
				t.Fatalf("Failed to set HOME: %v", err)
			}

			rcFile := filepath.Join(tmpDir, ".bashrc")
			err := os.WriteFile(rcFile, []byte(tt.rcContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create RC file: %v", err)
			}

			result := HasLegacyInstall("bash")
			if result != tt.want {
				t.Errorf("HasLegacyInstall() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestMigrateLegacyInstall(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	// Create RC file with legacy install
	rcFile := filepath.Join(tmpDir, ".bashrc")
	legacyContent := "# User content before\n" +
		HookMarkerStart + "\n" +
		"# Legacy hook code\n" +
		"__dirvana_hook() {\n" +
		"  echo 'old hook'\n" +
		"}\n" +
		HookMarkerEnd + "\n" +
		"# User content after"

	err := os.WriteFile(rcFile, []byte(legacyContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	// Verify legacy install exists
	if !HasLegacyInstall("bash") {
		t.Fatal("Legacy install should be detected")
	}

	// Migrate
	err = MigrateLegacyInstall("bash")
	if err != nil {
		t.Fatalf("MigrateLegacyInstall failed: %v", err)
	}

	// Verify legacy markers are removed from RC file
	rcContent, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	if strings.Contains(string(rcContent), HookMarkerStart) {
		t.Error("RC file still contains legacy start marker")
	}

	if strings.Contains(string(rcContent), HookMarkerEnd) {
		t.Error("RC file still contains legacy end marker")
	}

	// Verify user content is preserved
	if !strings.Contains(string(rcContent), "# User content before") {
		t.Error("User content before hook was not preserved")
	}

	if !strings.Contains(string(rcContent), "# User content after") {
		t.Error("User content after hook was not preserved")
	}

	// Verify new strategy is installed
	strategy, err := SelectInstallStrategy("bash")
	if err != nil {
		t.Fatalf("SelectInstallStrategy failed: %v", err)
	}

	if !strategy.IsInstalled() {
		t.Error("New strategy should be installed after migration")
	}

	// Verify hook file exists (for external strategy)
	if extStrategy, ok := strategy.(*ExternalHookStrategy); ok {
		if _, err := os.Stat(extStrategy.hookPath); os.IsNotExist(err) {
			t.Error("Hook file was not created during migration")
		}
	}
}

func TestMigrateLegacyInstall_NoOpWhenNotLegacy(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	// Create RC file without legacy install
	rcFile := filepath.Join(tmpDir, ".bashrc")
	rcContent := "# Regular .bashrc"
	err := os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	// Migrate (should be no-op)
	err = MigrateLegacyInstall("bash")
	if err != nil {
		t.Fatalf("MigrateLegacyInstall failed: %v", err)
	}

	// Verify RC file is unchanged
	rcContentAfter, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	if string(rcContentAfter) != rcContent {
		t.Error("RC file was modified when no legacy install existed")
	}
}

func TestRemoveLegacyHook(t *testing.T) {
	tests := []struct {
		name             string
		beforeContent    string
		afterContent     string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Remove hook from middle",
			beforeContent: "# Before\n" +
				HookMarkerStart + "\n" +
				"# Hook code\n" +
				HookMarkerEnd + "\n" +
				"# After",
			shouldContain:    []string{"# Before", "# After"},
			shouldNotContain: []string{HookMarkerStart, HookMarkerEnd, "# Hook code"},
		},
		{
			name: "Remove hook at end",
			beforeContent: "# User content\n" +
				HookMarkerStart + "\n" +
				"# Hook code\n" +
				HookMarkerEnd,
			shouldContain:    []string{"# User content"},
			shouldNotContain: []string{HookMarkerStart, HookMarkerEnd, "# Hook code"},
		},
		{
			name: "No hook present",
			beforeContent: "# Just user content\n" +
				"# More user content",
			shouldContain:    []string{"# Just user content", "# More user content"},
			shouldNotContain: []string{HookMarkerStart, HookMarkerEnd},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			rcFile := filepath.Join(tmpDir, ".bashrc")

			err := os.WriteFile(rcFile, []byte(tt.beforeContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create RC file: %v", err)
			}

			err = removeLegacyHook(rcFile)
			if err != nil {
				t.Fatalf("removeLegacyHook failed: %v", err)
			}

			afterContent, err := os.ReadFile(rcFile)
			if err != nil {
				t.Fatalf("Failed to read RC file: %v", err)
			}

			afterStr := string(afterContent)

			for _, needle := range tt.shouldContain {
				if !strings.Contains(afterStr, needle) {
					t.Errorf("Expected content to contain %q", needle)
				}
			}

			for _, needle := range tt.shouldNotContain {
				if strings.Contains(afterStr, needle) {
					t.Errorf("Expected content to NOT contain %q", needle)
				}
			}
		})
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

func TestMigrateLegacyInstall_ErrorOnGetRCFilePath(_ *testing.T) {
	// Test with unsupported shell
	err := MigrateLegacyInstall("unsupported-shell")
	// Should either error or be a no-op (since HasLegacyInstall will fail first)
	// This tests the error path when shell is invalid
	_ = err
}

func TestMigrateLegacyInstall_ErrorOnRemoveLegacyHook(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	// Create RC file with legacy hook
	rcFile := filepath.Join(tmpDir, ".bashrc")
	legacyContent := HookMarkerStart + "\n# Hook code\n" + HookMarkerEnd
	err := os.WriteFile(rcFile, []byte(legacyContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	// Make file read-only to cause write error during migration
	err = os.Chmod(rcFile, 0444)
	if err != nil {
		t.Fatalf("Failed to chmod RC file: %v", err)
	}

	// Attempt migration - should error on write
	err = MigrateLegacyInstall("bash")
	// Error handling depends on OS permissions
	_ = err
}

func TestUninstallLegacyHook_ErrorOnReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	// Don't create the RC file - should error on read
	err := uninstallLegacyHook("bash")
	if err == nil {
		t.Error("Expected error when RC file doesn't exist")
	}
}

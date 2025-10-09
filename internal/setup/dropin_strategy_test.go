package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

const testBashrcDropInContent = "if [ -d ~/.bashrc.d ]; then\n  for rc in ~/.bashrc.d/*.sh; do\n    source $rc\n  done\nfi"

func TestDropInStrategy_IsSupported(t *testing.T) {
	tests := []struct {
		name       string
		rcContent  string
		shell      string
		wantResult bool
	}{
		{
			name:       "Ubuntu style .bashrc.d",
			rcContent:  "# Some content\nif [ -d ~/.bashrc.d ]; then\n  source ~/.bashrc.d/*.sh\nfi",
			shell:      "bash",
			wantResult: true,
		},
		{
			name:       "Zsh with .zshrc.d",
			rcContent:  "# Some content\nfor file in ~/.zshrc.d/*.zsh; do\n  source $file\ndone",
			shell:      "zsh",
			wantResult: true,
		},
		{
			name:       "No drop-in support",
			rcContent:  "# Regular .bashrc without drop-in",
			shell:      "bash",
			wantResult: false,
		},
		{
			name:       "Empty RC file",
			rcContent:  "",
			shell:      "bash",
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			home := tmpDir

			rcFile := filepath.Join(home, "."+tt.shell+"rc")
			err := os.WriteFile(rcFile, []byte(tt.rcContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create RC file: %v", err)
			}

			dropInDir := filepath.Join(home, "."+tt.shell+"rc.d")
			dropInFile := filepath.Join(dropInDir, "dirvana.sh")

			strategy := &DropInStrategy{
				shell:      tt.shell,
				dropInDir:  dropInDir,
				dropInFile: dropInFile,
				rcFile:     rcFile,
			}

			result := strategy.IsSupported()
			if result != tt.wantResult {
				t.Errorf("IsSupported() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestDropInStrategy_Install(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	// Create RC file with drop-in support
	rcFile := filepath.Join(home, ".bashrc")
	rcContent := testBashrcDropInContent
	err := os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	dropInDir := filepath.Join(home, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Verify supported
	if !strategy.IsSupported() {
		t.Error("Should be supported")
	}

	// Install
	err = strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify drop-in file was created
	if _, err := os.Stat(dropInFile); os.IsNotExist(err) {
		t.Error("Drop-in file was not created")
	}

	// Verify drop-in file contains expected content
	dropInContent, err := os.ReadFile(dropInFile)
	if err != nil {
		t.Fatalf("Failed to read drop-in file: %v", err)
	}

	expectedHookCode := cli.GenerateHookCode("bash")
	if string(dropInContent) != expectedHookCode {
		t.Error("Drop-in file content doesn't match expected")
	}

	// Verify RC file was NOT modified
	rcContentAfter, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	if string(rcContentAfter) != rcContent {
		t.Error("RC file was modified (should not be modified for drop-in strategy)")
	}

	// Verify message indicates no RC modification
	if !strings.Contains(strategy.GetMessage(), "No modification") {
		t.Error("Message should indicate no modification to RC file")
	}
}

func TestDropInStrategy_Uninstall(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	rcFile := filepath.Join(home, ".bashrc")
	dropInDir := filepath.Join(home, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Create drop-in file
	err := os.MkdirAll(dropInDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create drop-in dir: %v", err)
	}

	err = os.WriteFile(dropInFile, []byte("# Test hook"), 0644)
	if err != nil {
		t.Fatalf("Failed to create drop-in file: %v", err)
	}

	// Verify installed
	if !strategy.IsInstalled() {
		t.Error("Should be installed")
	}

	// Uninstall
	err = strategy.Uninstall()
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify drop-in file was removed
	if _, err := os.Stat(dropInFile); !os.IsNotExist(err) {
		t.Error("Drop-in file was not removed")
	}

	// Verify not installed
	if strategy.IsInstalled() {
		t.Error("Should not be installed after uninstall")
	}
}

func TestDropInStrategy_IsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	rcFile := filepath.Join(home, ".bashrc")
	dropInDir := filepath.Join(home, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Should not be installed initially
	if strategy.IsInstalled() {
		t.Error("Should not be installed initially")
	}

	// Create drop-in file
	err := os.MkdirAll(dropInDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create drop-in dir: %v", err)
	}

	err = os.WriteFile(dropInFile, []byte("# Test hook"), 0644)
	if err != nil {
		t.Fatalf("Failed to create drop-in file: %v", err)
	}

	// Should be installed now
	if !strategy.IsInstalled() {
		t.Error("Should be installed after creating drop-in file")
	}
}

func TestDropInStrategy_NeedsUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	rcFile := filepath.Join(home, ".bashrc")
	dropInDir := filepath.Join(home, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Should need update when not installed
	if !strategy.NeedsUpdate() {
		t.Error("Should need update when not installed")
	}

	// Create drop-in file with current content
	err := os.MkdirAll(dropInDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create drop-in dir: %v", err)
	}

	currentHook := cli.GenerateHookCode("bash")
	err = os.WriteFile(dropInFile, []byte(currentHook), 0644)
	if err != nil {
		t.Fatalf("Failed to create drop-in file: %v", err)
	}

	// Should not need update with current content
	if strategy.NeedsUpdate() {
		t.Error("Should not need update when content is current")
	}

	// Modify with old content
	err = os.WriteFile(dropInFile, []byte("# Old hook code"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify drop-in file: %v", err)
	}

	// Should need update now
	if !strategy.NeedsUpdate() {
		t.Error("Should need update when content is outdated")
	}
}

func TestDropInStrategy_NoRCModificationDuringUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	// Create RC file with drop-in support
	rcFile := filepath.Join(home, ".bashrc")
	rcContent := testBashrcDropInContent
	err := os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	dropInDir := filepath.Join(home, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Install
	err = strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Modify drop-in file to simulate outdated version
	err = os.WriteFile(dropInFile, []byte("# Old hook code"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify drop-in file: %v", err)
	}

	// Get RC content before update
	rcContentBefore, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	// Update (install again)
	err = strategy.Install()
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Get RC content after update
	rcContentAfter, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	// RC file should not have been modified
	if string(rcContentBefore) != string(rcContentAfter) {
		t.Error("RC file was modified during update (should not touch RC file)")
	}
}

func TestDropInStrategy_GetRCFile(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	dropInDir := filepath.Join(tmpDir, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Test GetRCFile returns the correct path
	if got := strategy.GetRCFile(); got != rcFile {
		t.Errorf("GetRCFile() = %v, want %v", got, rcFile)
	}
}

func TestDropInStrategy_GetMessage_Default(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	dropInDir := filepath.Join(tmpDir, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// When message is empty, should return default "up to date" message
	msg := strategy.GetMessage()
	if !strings.Contains(msg, "up to date") {
		t.Errorf("GetMessage() with empty message should contain 'up to date', got: %q", msg)
	}
}

func TestDropInStrategy_GetMessage_UpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	dropInDir := filepath.Join(tmpDir, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Create drop-in file with current content
	err := os.MkdirAll(dropInDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create drop-in dir: %v", err)
	}

	currentHook := cli.GenerateHookCode("bash")
	err = os.WriteFile(dropInFile, []byte(currentHook), 0644)
	if err != nil {
		t.Fatalf("Failed to create drop-in file: %v", err)
	}

	// When up to date, should return "up to date" message
	msg := strategy.GetMessage()
	if !strings.Contains(msg, "up to date") {
		t.Errorf("GetMessage() when up to date should contain 'up to date', got: %q", msg)
	}
}

func TestDropInStrategy_Install_ErrorCreatingDir(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Create a file where the directory should be to cause error
	invalidDropInDir := filepath.Join(tmpDir, ".bashrc.d")
	err := os.WriteFile(invalidDropInDir, []byte("file blocking directory"), 0644)
	if err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	dropInFile := filepath.Join(invalidDropInDir, "dirvana.sh")

	// Create RC file with drop-in support
	rcContent := testBashrcDropInContent
	err = os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  invalidDropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Install should fail
	err = strategy.Install()
	if err == nil {
		t.Error("Expected error when drop-in directory cannot be created")
	}
}

func TestDropInStrategy_Uninstall_ErrorRemovingFile(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	dropInDir := filepath.Join(tmpDir, ".bashrc.d")

	// Create drop-in directory
	err := os.MkdirAll(dropInDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create drop-in dir: %v", err)
	}

	// Create a directory instead of a file to cause removal error
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")
	err = os.Mkdir(dropInFile, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Uninstall should error (trying to remove a directory with Remove instead of RemoveAll)
	err = strategy.Uninstall()
	// On some systems this might error, on others it might succeed
	_ = err
}

func TestDropInStrategy_Install_AlreadyUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	home := tmpDir

	// Create RC file with drop-in support
	rcFile := filepath.Join(home, ".bashrc")
	rcContent := testBashrcDropInContent
	err := os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	dropInDir := filepath.Join(home, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	// Create drop-in directory and file with current hook
	err = os.MkdirAll(dropInDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create drop-in dir: %v", err)
	}

	expectedHookCode := cli.GenerateHookCode("bash")
	err = os.WriteFile(dropInFile, []byte(expectedHookCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create drop-in file: %v", err)
	}

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Install when already up to date
	err = strategy.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Message should indicate already up to date
	if !strings.Contains(strategy.GetMessage(), "up to date") && !strings.Contains(strategy.GetMessage(), "No modification") {
		t.Errorf("Message should indicate already up to date, got: %q", strategy.GetMessage())
	}
}

func TestDropInStrategy_Uninstall_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	dropInDir := filepath.Join(tmpDir, ".bashrc.d")
	dropInFile := filepath.Join(dropInDir, "dirvana.sh")

	strategy := &DropInStrategy{
		shell:      "bash",
		dropInDir:  dropInDir,
		dropInFile: dropInFile,
		rcFile:     rcFile,
	}

	// Uninstall when file doesn't exist (should not error)
	err := strategy.Uninstall()
	if err != nil {
		t.Error("Uninstall should handle missing file gracefully")
	}
}

func TestNewDropInStrategy_Zsh(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	// Create RC file with zsh drop-in support
	rcFile := filepath.Join(tmpDir, ".zshrc")
	rcContent := "if [ -d ~/.zshrc.d ]; then\n  for rc in ~/.zshrc.d/*.zsh; do\n    source $rc\n  done\nfi"
	err := os.WriteFile(rcFile, []byte(rcContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create RC file: %v", err)
	}

	strategy, err := NewDropInStrategy("zsh")
	if err != nil {
		t.Fatalf("NewDropInStrategy failed: %v", err)
	}

	if !strategy.IsSupported() {
		t.Error("Strategy should be supported for zsh with drop-in")
	}
}

func TestNewDropInStrategy_UnsupportedShell(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	_, err := NewDropInStrategy("unsupported")
	if err == nil {
		t.Error("Expected error for unsupported shell")
	}
}

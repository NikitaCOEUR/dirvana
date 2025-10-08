package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/cli"
)

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
	rcContent := "if [ -d ~/.bashrc.d ]; then\n  for rc in ~/.bashrc.d/*.sh; do\n    source $rc\n  done\nfi"
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
	rcContent := "if [ -d ~/.bashrc.d ]; then\n  for rc in ~/.bashrc.d/*.sh; do\n    source $rc\n  done\nfi"
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

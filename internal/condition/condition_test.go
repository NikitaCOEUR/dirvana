package condition

import (
	"os"
	"path/filepath"
	"testing"
)

// resolveSymlinks resolves symlinks to get the real path (needed for macOS where /tmp -> /private/tmp)
func resolveSymlinks(t *testing.T, path string) string {
	t.Helper()
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("Failed to resolve symlinks for %s: %v", path, err)
	}
	return realPath
}

// TestFileCondition tests the FileCondition evaluation
func TestFileCondition(t *testing.T) {
	// Create temp directory for testing
	// Resolve symlinks for macOS compatibility where /tmp -> /private/tmp
	tmpDir := resolveSymlinks(t, t.TempDir())

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory (not a file)
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		ctx     Context
		wantOk  bool
		wantMsg string
	}{
		{
			name: "file exists - absolute path",
			path: testFile,
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk: true,
		},
		{
			name: "file exists - relative path",
			path: "test.txt",
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk: true,
		},
		{
			name: "file does not exist",
			path: filepath.Join(tmpDir, "nonexistent.txt"),
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk:  false,
			wantMsg: "does not exist",
		},
		{
			name: "path is directory not file",
			path: testDir,
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk:  false,
			wantMsg: "is a directory, not a file",
		},
		{
			name: "file with env var expansion",
			path: "$TEST_FILE",
			ctx: Context{
				Env:        map[string]string{"TEST_FILE": testFile},
				WorkingDir: tmpDir,
			},
			wantOk: true,
		},
		{
			name: "file with undefined env var",
			path: "$UNDEFINED_VAR",
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk:  false,
			wantMsg: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := FileCondition{Path: tt.path}
			ok, msg, err := cond.Evaluate(tt.ctx)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if ok != tt.wantOk {
				t.Errorf("Expected ok=%v, got %v", tt.wantOk, ok)
			}

			if !ok && tt.wantMsg != "" {
				if msg == "" {
					t.Errorf("Expected error message containing '%s', got empty", tt.wantMsg)
				}
			}
		})
	}
}

// TestVarCondition tests the VarCondition evaluation
func TestVarCondition(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		ctx     Context
		wantOk  bool
	}{
		{
			name:    "var exists and non-empty",
			varName: "TEST_VAR",
			ctx: Context{
				Env: map[string]string{"TEST_VAR": "value"},
			},
			wantOk: true,
		},
		{
			name:    "var does not exist",
			varName: "NONEXISTENT",
			ctx: Context{
				Env: map[string]string{},
			},
			wantOk: false,
		},
		{
			name:    "var exists but empty",
			varName: "EMPTY_VAR",
			ctx: Context{
				Env: map[string]string{"EMPTY_VAR": ""},
			},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := VarCondition{Name: tt.varName}
			ok, _, err := cond.Evaluate(tt.ctx)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if ok != tt.wantOk {
				t.Errorf("Expected ok=%v, got %v", tt.wantOk, ok)
			}
		})
	}
}

// TestDirCondition tests the DirCondition evaluation
func TestDirCondition(t *testing.T) {
	// Resolve symlinks for macOS compatibility
	tmpDir := resolveSymlinks(t, t.TempDir())

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create a test file (not a directory)
	testFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		ctx     Context
		wantOk  bool
		wantMsg string
	}{
		{
			name: "directory exists - absolute path",
			path: testDir,
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk: true,
		},
		{
			name: "directory exists - relative path",
			path: "testdir",
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk: true,
		},
		{
			name: "directory does not exist",
			path: filepath.Join(tmpDir, "nonexistent"),
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk:  false,
			wantMsg: "does not exist",
		},
		{
			name: "path is file not directory",
			path: testFile,
			ctx: Context{
				Env:        map[string]string{},
				WorkingDir: tmpDir,
			},
			wantOk:  false,
			wantMsg: "is a file, not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := DirCondition{Path: tt.path}
			ok, msg, err := cond.Evaluate(tt.ctx)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if ok != tt.wantOk {
				t.Errorf("Expected ok=%v, got %v", tt.wantOk, ok)
			}

			if !ok && tt.wantMsg != "" {
				if msg == "" {
					t.Errorf("Expected error message containing '%s', got empty", tt.wantMsg)
				}
			}
		})
	}
}

// TestCommandCondition tests the CommandCondition evaluation
func TestCommandCondition(t *testing.T) {
	tests := []struct {
		name    string
		cmdName string
		wantOk  bool
	}{
		{
			name:    "command exists (sh)",
			cmdName: "sh",
			wantOk:  true,
		},
		{
			name:    "command does not exist",
			cmdName: "this-command-definitely-does-not-exist",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := CommandCondition{Name: tt.cmdName}
			ok, _, err := cond.Evaluate(Context{})

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if ok != tt.wantOk {
				t.Errorf("Expected ok=%v, got %v", tt.wantOk, ok)
			}
		})
	}
}

// TestAllCondition tests the AllCondition (AND logic)
func TestAllCondition(t *testing.T) {
	// Resolve symlinks for macOS compatibility
	tmpDir := resolveSymlinks(t, t.TempDir())
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := Context{
		Env:        map[string]string{"VAR1": "value1"},
		WorkingDir: tmpDir,
	}

	tests := []struct {
		name       string
		conditions []Condition
		wantOk     bool
	}{
		{
			name: "all conditions pass",
			conditions: []Condition{
				FileCondition{Path: "test.txt"},
				VarCondition{Name: "VAR1"},
			},
			wantOk: true,
		},
		{
			name: "one condition fails",
			conditions: []Condition{
				FileCondition{Path: "test.txt"},
				VarCondition{Name: "NONEXISTENT"},
			},
			wantOk: false,
		},
		{
			name: "all conditions fail",
			conditions: []Condition{
				FileCondition{Path: "nonexistent.txt"},
				VarCondition{Name: "NONEXISTENT"},
			},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := AllCondition{Conditions: tt.conditions}
			ok, _, err := cond.Evaluate(ctx)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if ok != tt.wantOk {
				t.Errorf("Expected ok=%v, got %v", tt.wantOk, ok)
			}
		})
	}
}

// TestAnyCondition tests the AnyCondition (OR logic)
func TestAnyCondition(t *testing.T) {
	// Resolve symlinks for macOS compatibility
	tmpDir := resolveSymlinks(t, t.TempDir())
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := Context{
		Env:        map[string]string{"VAR1": "value1"},
		WorkingDir: tmpDir,
	}

	tests := []struct {
		name       string
		conditions []Condition
		wantOk     bool
	}{
		{
			name: "all conditions pass",
			conditions: []Condition{
				FileCondition{Path: "test.txt"},
				VarCondition{Name: "VAR1"},
			},
			wantOk: true,
		},
		{
			name: "one condition passes",
			conditions: []Condition{
				FileCondition{Path: "test.txt"},
				VarCondition{Name: "NONEXISTENT"},
			},
			wantOk: true,
		},
		{
			name: "all conditions fail",
			conditions: []Condition{
				FileCondition{Path: "nonexistent.txt"},
				VarCondition{Name: "NONEXISTENT"},
			},
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := AnyCondition{Conditions: tt.conditions}
			ok, _, err := cond.Evaluate(ctx)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if ok != tt.wantOk {
				t.Errorf("Expected ok=%v, got %v", tt.wantOk, ok)
			}
		})
	}
}

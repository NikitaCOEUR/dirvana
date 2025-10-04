package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
)

// TestExport_HierarchyWithUnauthorizedMiddle tests config loading in hierarchy A/B/C
// where B is unauthorized but has no local config, so A and C should be loaded
//nolint:gocyclo // Test function with multiple scenarios
func TestExport_HierarchyWithUnauthorizedMiddle(t *testing.T) {
	// Save and restore current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("Warning: failed to restore original directory: %v", err)
		}
	}()

	// Setup test directories
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "A")
	dirB := filepath.Join(dirA, "B")
	dirC := filepath.Join(dirB, "C")

	// Create directory structure
	for _, dir := range []string{dirA, dirB, dirC} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create config files
	configA := `aliases:
  cmd_a: echo "from A"
`
	configB := `aliases:
  cmd_b: echo "from B"
`
	configC := `aliases:
  cmd_c: echo "from C"
`

	if err := os.WriteFile(filepath.Join(dirA, ".dirvana.yml"), []byte(configA), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, ".dirvana.yml"), []byte(configB), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirC, ".dirvana.yml"), []byte(configC), 0644); err != nil {
		t.Fatal(err)
	}

	// Setup auth and cache
	authPath := filepath.Join(tmpDir, "auth.json")
	cachePath := filepath.Join(tmpDir, "cache.json")

	authMgr, err := auth.New(authPath)
	if err != nil {
		t.Fatal(err)
	}

	// Authorize A and C, but NOT B
	if err := authMgr.Allow(dirA); err != nil {
		t.Fatal(err)
	}
	if err := authMgr.Allow(dirC); err != nil {
		t.Fatal(err)
	}

	// Test 1: Navigate to A
	if err := os.Chdir(dirA); err != nil {
		t.Fatal(err)
	}

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	output := captureOutput(t, func() error {
		return Export(params)
	})

	// Should have cmd_a
	if !strings.Contains(output, "cmd_a") {
		t.Errorf("Expected cmd_a in A, got:\n%s", output)
	}
	if strings.Contains(output, "cmd_b") {
		t.Errorf("Should not have cmd_b in A (B is not authorized)")
	}
	if strings.Contains(output, "cmd_c") {
		t.Errorf("Should not have cmd_c yet (not in C)")
	}

	// Test 2: Navigate to B (not authorized, should still have A)
	if err := os.Chdir(dirB); err != nil {
		t.Fatal(err)
	}

	params.PrevDir = dirA
	output = captureOutput(t, func() error {
		return Export(params)
	})

	// Should still have cmd_a (inherited from A)
	// Should NOT have cmd_b (B not authorized)
	if !strings.Contains(output, "cmd_a") {
		t.Errorf("Expected cmd_a in B (inherited from A), got:\n%s", output)
	}
	if strings.Contains(output, "cmd_b") {
		t.Errorf("Should not have cmd_b in B (not authorized)")
	}

	// Test 3: Navigate to C (should have A and C, not B)
	if err := os.Chdir(dirC); err != nil {
		t.Fatal(err)
	}

	params.PrevDir = dirB
	output = captureOutput(t, func() error {
		return Export(params)
	})

	// Should have cmd_a (from A) and cmd_c (from C), but not cmd_b
	if !strings.Contains(output, "cmd_a") {
		t.Errorf("Expected cmd_a in C (inherited from A), got:\n%s", output)
	}
	if !strings.Contains(output, "cmd_c") {
		t.Errorf("Expected cmd_c in C, got:\n%s", output)
	}
	if strings.Contains(output, "cmd_b") {
		t.Errorf("Should not have cmd_b in C (B not authorized)")
	}

	// Test 4: Navigate back to B (THIS IS THE KEY TEST - should cleanup cmd_c)
	if err := os.Chdir(dirB); err != nil {
		t.Fatal(err)
	}

	params.PrevDir = dirC
	output = captureOutput(t, func() error {
		return Export(params)
	})

	// Should cleanup cmd_c and keep cmd_a
	if strings.Contains(output, "unalias cmd_c") {
		t.Logf("✓ Correctly cleaning up cmd_c when leaving C")
	} else {
		t.Errorf("Should cleanup cmd_c when going from C to B, got:\n%s", output)
	}

	if !strings.Contains(output, "cmd_a") {
		t.Errorf("Expected cmd_a in B (inherited from A), got:\n%s", output)
	}
	if strings.Contains(output, "alias cmd_c=") {
		t.Errorf("Should not define cmd_c in B (left C), got:\n%s", output)
	}
	if strings.Contains(output, "cmd_b") {
		t.Errorf("Should not have cmd_b in B (not authorized)")
	}

	// Test 5: Navigate to parent of A (should cleanup everything)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	params.PrevDir = dirB
	output = captureOutput(t, func() error {
		return Export(params)
	})

	// Should cleanup cmd_a
	if strings.Contains(output, "unalias cmd_a") {
		t.Logf("✓ Correctly cleaning up cmd_a when leaving A hierarchy")
	} else {
		t.Errorf("Should cleanup cmd_a when leaving A hierarchy, got:\n%s", output)
	}
}

// TestExport_LocalOnlyFlag tests that local_only flag prevents loading parent configs
//nolint:gocyclo // Test function with multiple scenarios
func TestExport_LocalOnlyFlag(t *testing.T) {
	// Save and restore current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("Warning: failed to restore original directory: %v", err)
		}
	}()

	// Setup test directories
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "A")
	dirB := filepath.Join(dirA, "B")
	dirC := filepath.Join(dirB, "C")
	dirE := filepath.Join(tmpDir, "E")
	dirF := filepath.Join(dirE, "F")
	dirG := filepath.Join(dirF, "G")

	// Create directory structures
	for _, dir := range []string{dirA, dirB, dirC, dirE, dirF, dirG} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create config files for A/B/C
	configA := `aliases:
  cmd_a: echo "from A"
`
	configB := `aliases:
  cmd_b: echo "from B"
`
	configC := `aliases:
  cmd_c: echo "from C"
`

	// Create configs for E/F/G with G having local_only
	configE := `aliases:
  cmd_e: echo "from E"
`
	configF := `aliases:
  cmd_f: echo "from F"
`
	configG := `local_only: true
aliases:
  cmd_g: echo "from G"
`

	for _, cfg := range []struct {
		path    string
		content string
	}{
		{filepath.Join(dirA, ".dirvana.yml"), configA},
		{filepath.Join(dirB, ".dirvana.yml"), configB},
		{filepath.Join(dirC, ".dirvana.yml"), configC},
		{filepath.Join(dirE, ".dirvana.yml"), configE},
		{filepath.Join(dirF, ".dirvana.yml"), configF},
		{filepath.Join(dirG, ".dirvana.yml"), configG},
	} {
		if err := os.WriteFile(cfg.path, []byte(cfg.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Setup auth and cache
	authPath := filepath.Join(tmpDir, "auth.json")
	cachePath := filepath.Join(tmpDir, "cache.json")

	authMgr, err := auth.New(authPath)
	if err != nil {
		t.Fatal(err)
	}

	// Authorize all directories
	for _, dir := range []string{dirA, dirB, dirC, dirE, dirF, dirG} {
		if err := authMgr.Allow(dir); err != nil {
			t.Fatal(err)
		}
	}

	// Navigate to C
	if err := os.Chdir(dirC); err != nil {
		t.Fatal(err)
	}

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	output := captureOutput(t, func() error {
		return Export(params)
	})

	// Should have all A, B, C
	if !strings.Contains(output, "cmd_a") || !strings.Contains(output, "cmd_b") || !strings.Contains(output, "cmd_c") {
		t.Errorf("Expected cmd_a, cmd_b, cmd_c in C, got:\n%s", output)
	}

	// Navigate to G (which has local_only)
	if err := os.Chdir(dirG); err != nil {
		t.Fatal(err)
	}

	params.PrevDir = dirC
	output = captureOutput(t, func() error {
		return Export(params)
	})

	// Should cleanup A, B, C
	if !strings.Contains(output, "unalias cmd_a") || !strings.Contains(output, "unalias cmd_b") || !strings.Contains(output, "unalias cmd_c") {
		t.Errorf("Should cleanup cmd_a, cmd_b, cmd_c when entering G with local_only, got:\n%s", output)
	}

	// Should ONLY have cmd_g (due to local_only)
	if !strings.Contains(output, "cmd_g") {
		t.Errorf("Expected cmd_g in G, got:\n%s", output)
	}
	if strings.Contains(output, "alias cmd_e=") || strings.Contains(output, "alias cmd_f=") {
		t.Errorf("Should not have cmd_e or cmd_f in G (local_only), got:\n%s", output)
	}
}

// captureOutput captures stdout during function execution
func captureOutput(t *testing.T, fn func() error) string {
	t.Helper()

	// Create temp file for output
	tmpfile, err := os.CreateTemp("", "output")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Redirect stdout to temp file
	os.Stdout = tmpfile
	err = fn()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Function failed: %v", err)
	}

	// Read output
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

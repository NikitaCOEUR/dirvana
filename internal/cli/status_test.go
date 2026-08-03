package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAliasConfig = `aliases:
  test: echo test
`

const childAliasConfig = `aliases:
  child: echo child
`

func TestStatus_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error even with no config
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_NotAuthorized(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte(testAliasConfig), 0o644)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error but should show not authorized
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_Authorized(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
  ll: ls -la
functions:
  greet: echo "Hello, $1!"
env:
  PROJECT_NAME: myproject
  GIT_BRANCH:
    sh: git branch --show-current
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and display status
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  ll: ls -la
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	// First export to populate cache
	exportParams := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}
	err = Export(exportParams)
	require.NoError(t, err)

	// Now check status - should show cache hit
	statusParams := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}
	err = Status(statusParams)
	require.NoError(t, err)
}

func TestStatus_WithHierarchy(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create parent and child directories
	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(tmpDir, "parent", "child")
	require.NoError(t, os.MkdirAll(childDir, 0o755))

	// Create parent config
	parentConfig := filepath.Join(parentDir, ".dirvana.yml")
	parentContent := `aliases:
  parent: echo parent
`
	require.NoError(t, os.WriteFile(parentConfig, []byte(parentContent), 0o644))

	// Create child config
	childConfig := filepath.Join(childDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(childConfig, []byte(childAliasConfig), 0o644))

	// Authorize both directories
	require.NoError(t, Allow(authPath, parentDir))
	require.NoError(t, Allow(authPath, childDir))

	// Change to child dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(childDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and show hierarchy
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithFlags(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with flags
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
local_only: true
ignore_global: true
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and display flags
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithLongAlias(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with a very long alias command
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	longCommand := "echo this is a very long command that should be truncated when displayed in the status output to avoid cluttering the terminal"
	configContent := "aliases:\n  longcmd: " + longCommand + "\n"
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and truncate the long alias
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithAdvancedAliases(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with advanced alias configurations
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  # Simple alias
  simple: echo test

  # Advanced alias with completion disabled
  nocomp:
    command: echo no completion
    completion: false

  # Advanced alias with inherited completion
  withcomp:
    command: kubectl get pods
    completion: kubectl
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and handle different alias types
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_InvalidCachePath(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := "/invalid/path/that/does/not/exist/cache.json"
	authPath := filepath.Join(tmpDir, "auth.json")

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte("aliases:\n  test: echo test\n"), 0o644)
	require.NoError(t, err)
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := StatusParams{CachePath: cachePath, AuthPath: authPath}
	err = Status(params)
	require.Error(t, err)
}

func TestStatus_InvalidAuthPath(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := "/invalid/path/that/does/not/exist/auth.json"

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	params := StatusParams{CachePath: cachePath, AuthPath: authPath}
	err = Status(params)
	require.Error(t, err)
}

func TestStatus_WithMixedAuthorizations(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create directory hierarchy: A/B/C
	dirA := filepath.Join(tmpDir, "A")
	dirB := filepath.Join(dirA, "B")
	dirC := filepath.Join(dirB, "C")
	require.NoError(t, os.MkdirAll(dirC, 0o755))

	// Create configs in each directory
	configA := filepath.Join(dirA, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configA, []byte("aliases:\n  a: echo a\n"), 0o644))

	configB := filepath.Join(dirB, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configB, []byte("aliases:\n  b: echo b\n"), 0o644))

	configC := filepath.Join(dirC, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configC, []byte("aliases:\n  c: echo c\n"), 0o644))

	// Authorize only A and C, not B
	require.NoError(t, Allow(authPath, dirA))
	require.NoError(t, Allow(authPath, dirC))

	// Change to dirC
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(dirC))

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should show authorization status for each config
	err = Status(params)
	require.NoError(t, err)
}

func TestStatus_WithLocalOnly(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create directory hierarchy: parent/child
	dirParent := filepath.Join(tmpDir, "parent")
	dirChild := filepath.Join(dirParent, "child")
	require.NoError(t, os.MkdirAll(dirChild, 0o755))

	// Create parent config
	configParent := filepath.Join(dirParent, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configParent, []byte("aliases:\n  parent: echo parent\n"), 0o644))

	// Create child config with local_only
	configChild := filepath.Join(dirChild, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configChild, []byte("local_only: true\naliases:\n  child: echo child\n"), 0o644))

	// Authorize both directories
	require.NoError(t, Allow(authPath, dirParent))
	require.NoError(t, Allow(authPath, dirChild))

	// Change to dirChild
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(dirChild))

	params := StatusParams{
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should only show child config due to local_only
	err = Status(params)
	require.NoError(t, err)
}

// TestStatus_PlainOnATerminal proves --plain wins over the terminal check: the
// report is printed and the foldable view never opens, which is what a user
// piping the output through a pager or copying it wants.
func TestStatus_PlainOnATerminal(t *testing.T) {
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	// Pretend both ends are a terminal; without --plain this would try to take
	// it over and hang the test
	original := isTerminal
	defer func() { isTerminal = original }()
	isTerminal = func(*os.File) bool { return true }

	err = Status(StatusParams{
		CachePath: filepath.Join(tmpDir, "cache.json"),
		AuthPath:  filepath.Join(tmpDir, "auth.json"),
		Plain:     true,
	})
	require.NoError(t, err)
}

// TestStatus_PlainWhenRedirected covers the default path taken by a pipe, a
// log or the test suite itself.
func TestStatus_PlainWhenRedirected(t *testing.T) {
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	original := isTerminal
	defer func() { isTerminal = original }()
	isTerminal = func(*os.File) bool { return false }

	err = Status(StatusParams{
		CachePath: filepath.Join(tmpDir, "cache.json"),
		AuthPath:  filepath.Join(tmpDir, "auth.json"),
	})
	require.NoError(t, err)
}

// TestStatus_DevNullIsNotATerminal pins the check that used to be a mode test:
// /dev/null has ModeCharDevice set, so `dirvana status > /dev/null` opened the
// interactive view, drew into the void and hung until interrupted.
func TestStatus_DevNullIsNotATerminal(t *testing.T) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer func() { _ = devNull.Close() }()

	assert.False(t, isTerminal(devNull), "%s must not pass for a terminal", os.DevNull)
}

// TestStatus_OpensTheFoldableViewOnATerminal covers the branch the other tests
// deliberately avoid, without a program taking over the terminal.
func TestStatus_OpensTheFoldableViewOnATerminal(t *testing.T) {
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	originalTerminal, originalRun := isTerminal, runInteractive
	defer func() { isTerminal, runInteractive = originalTerminal, originalRun }()

	isTerminal = func(*os.File) bool { return true }
	called := false
	runInteractive = func(data *status.Data, _ io.Reader, _ io.Writer) error {
		called = true
		assert.NotNil(t, data)
		return nil
	}

	require.NoError(t, Status(StatusParams{
		CachePath: filepath.Join(tmpDir, "cache.json"),
		AuthPath:  filepath.Join(tmpDir, "auth.json"),
	}))
	assert.True(t, called, "a terminal should get the foldable view")
}

func TestStatus_ReportsAnInteractiveFailure(t *testing.T) {
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	originalTerminal, originalRun := isTerminal, runInteractive
	defer func() { isTerminal, runInteractive = originalTerminal, originalRun }()

	isTerminal = func(*os.File) bool { return true }
	runInteractive = func(*status.Data, io.Reader, io.Writer) error { return assert.AnError }

	err = Status(StatusParams{
		CachePath: filepath.Join(tmpDir, "cache.json"),
		AuthPath:  filepath.Join(tmpDir, "auth.json"),
	})
	assert.ErrorIs(t, err, assert.AnError)
}

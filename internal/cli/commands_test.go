package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/NikitaCOEUR/dirvana/internal/core"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPathConst = "/test/path"

func TestAllow(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	testPath := testPathConst

	err := Allow(authPath, testPath)
	require.NoError(t, err)

	// Verify it was actually allowed
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	allowed, err := authMgr.IsAllowed(testPath)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestAllow_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Test with empty path - auth package allows it, so no error
	err := Allow(authPath, "")
	require.NoError(t, err)
}

func TestAllow_InvalidAuthPath(t *testing.T) {
	// Test with invalid auth path
	err := Allow("/invalid/nonexistent/dir/auth.json", "/test/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestRevoke(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	testPath := testPathConst

	// First allow
	err := Allow(authPath, testPath)
	require.NoError(t, err)

	// Then revoke
	err = Revoke(authPath, testPath)
	require.NoError(t, err)

	// Verify it was revoked
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)
	allowed, err := authMgr.IsAllowed(testPath)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRevoke_NotAuthorized(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	testPath := testPathConst

	// Try to revoke a path that was never authorized
	err := Revoke(authPath, testPath)
	// Should not error even if path wasn't authorized
	require.NoError(t, err)
}

func TestRevoke_InvalidAuthPath(t *testing.T) {
	// Test with invalid auth path
	err := Revoke("/invalid/nonexistent/dir/auth.json", "/test/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize auth")
}

func TestRevokeWithParams_FromRevokedDir(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	testDir := filepath.Join(tmpDir, "testdir")
	require.NoError(t, os.MkdirAll(testDir, 0o755))

	// First allow the test directory
	err := Allow(authPath, testDir)
	require.NoError(t, err)

	// Change to the test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(testDir))

	// Revoke while in the directory - should show cleanup tip
	err = RevokeWithParams(RevokeParams{
		AuthPath:     authPath,
		PathToRevoke: testDir,
	})
	require.NoError(t, err)
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")

	// Test with no authorized paths
	err := List(authPath)
	require.NoError(t, err)

	// Add some paths
	err = Allow(authPath, "/test/path1")
	require.NoError(t, err)
	err = Allow(authPath, "/test/path2")
	require.NoError(t, err)

	// Test with authorized paths
	err = List(authPath)
	require.NoError(t, err)
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run init
	err = Init(false)
	require.NoError(t, err)

	// Verify config file was created
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "yaml-language-server: $schema=")
	assert.Contains(t, content, "aliases:")
	assert.Contains(t, content, "functions:")
	assert.Contains(t, content, "env:")
	assert.Contains(t, content, "local_only:")
}

func TestInit_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create config file first
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte("test"), 0o644)
	require.NoError(t, err)

	// Run init should fail
	err = Init(false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInit_Global(t *testing.T) {
	// Override XDG_CONFIG_HOME to use temp dir
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Run init with global flag
	err := Init(true)
	require.NoError(t, err)

	// Verify global config file was created
	globalConfigPath := filepath.Join(tmpDir, "dirvana", "global.yml")
	data, err := os.ReadFile(globalConfigPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "yaml-language-server: $schema=")
	assert.Contains(t, content, "aliases:")
	assert.Contains(t, content, "functions:")
	assert.Contains(t, content, "env:")
	assert.Contains(t, content, "local_only:")
}

func TestInit_Global_AlreadyExists(t *testing.T) {
	// Override XDG_CONFIG_HOME to use temp dir
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config file first
	globalConfigDir := filepath.Join(tmpDir, "dirvana")
	err := os.MkdirAll(globalConfigDir, 0o755)
	require.NoError(t, err)

	globalConfigPath := filepath.Join(globalConfigDir, "global.yml")
	err = os.WriteFile(globalConfigPath, []byte("test"), 0o644)
	require.NoError(t, err)

	// Run init with global flag should fail
	err = Init(true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestExport_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error even with no config
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_NotAuthorized(t *testing.T) {
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

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should not error but should warn (we just test it doesn't crash)
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_Authorized(t *testing.T) {
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

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and generate shell code
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_CacheHit(t *testing.T) {
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

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// First call - should generate and cache
	err = Export(params)
	require.NoError(t, err)

	// Second call - should use cache
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_WithContextCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create parent and child directories
	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(tmpDir, "child")
	require.NoError(t, os.MkdirAll(parentDir, 0o755))
	require.NoError(t, os.MkdirAll(childDir, 0o755))

	// Create configs
	parentConfig := filepath.Join(parentDir, ".dirvana.yml")
	parentContent := `aliases:
  parent: echo parent
`
	require.NoError(t, os.WriteFile(parentConfig, []byte(parentContent), 0o644))

	childConfig := filepath.Join(childDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(childConfig, []byte(childAliasConfig), 0o644))

	// Authorize both directories
	require.NoError(t, Allow(authPath, parentDir))
	require.NoError(t, Allow(authPath, childDir))

	// Change to parent dir and export
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(parentDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	err = Export(params)
	require.NoError(t, err)

	// Change to child dir with previous dir set - should trigger cleanup
	err = os.Chdir(childDir)
	require.NoError(t, err)

	params.PrevDir = parentDir
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_WithShellEnv(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with shell-based env vars
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
env:
  STATIC_VAR: static
  GIT_BRANCH:
    sh: git branch --show-current
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Enable test mode to use stdin instead of /dev/tty
	oldTestMode := os.Getenv("DIRVANA_TEST_MODE")
	require.NoError(t, os.Setenv("DIRVANA_TEST_MODE", "1"))
	defer func() {
		if oldTestMode == "" {
			_ = os.Unsetenv("DIRVANA_TEST_MODE")
		} else {
			_ = os.Setenv("DIRVANA_TEST_MODE", oldTestMode)
		}
	}()

	// Simulate user approval for shell commands
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = w.WriteString("y\n")
	require.NoError(t, err)
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Should succeed and generate shell code with env vars
	err = Export(params)
	require.NoError(t, err)
}

func TestExport_WithFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Change to temp dir
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a config file with functions
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `functions:
  greet: |
    echo "Hello, $1!"
  mkcd: |
    mkdir -p "$1" && cd "$1"
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Should succeed and generate shell code with functions
	err = Export(params)
	require.NoError(t, err)
}

// TestDisplayShellCommandsForApproval tests the display of shell commands for user approval
func TestDisplayShellCommandsForApproval(t *testing.T) {
	t.Run("WithCommands", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		shellEnv := map[string]string{
			"GIT_BRANCH": "git rev-parse --abbrev-ref HEAD",
			"USER":       "whoami",
		}

		err := displayShellCommandsForApproval(shellEnv)
		_ = w.Close()
		os.Stderr = oldStderr

		require.NoError(t, err)

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "GIT_BRANCH: git rev-parse --abbrev-ref HEAD")
		assert.Contains(t, output, "USER: whoami")
		assert.Contains(t, output, "This configuration contains dynamic shell commands")
		assert.Contains(t, output, "These commands will execute to set environment variables.")
	})

	t.Run("Empty", func(t *testing.T) {
		// Should not print anything or error
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		err := displayShellCommandsForApproval(map[string]string{})
		_ = w.Close()
		os.Stderr = oldStderr

		require.NoError(t, err)

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		output := buf.String()
		assert.Empty(t, output)
	})
}

// TestPromptShellApproval tests the user approval prompt for shell commands
func TestPromptShellApproval(t *testing.T) {
	t.Run("Approved", func(t *testing.T) {
		// Enable test mode to use stdin instead of /dev/tty
		oldTestMode := os.Getenv("DIRVANA_TEST_MODE")
		require.NoError(t, os.Setenv("DIRVANA_TEST_MODE", "1"))
		defer func() {
			if oldTestMode == "" {
				_ = os.Unsetenv("DIRVANA_TEST_MODE")
			} else {
				_ = os.Setenv("DIRVANA_TEST_MODE", oldTestMode)
			}
		}()

		// Simulate user input "y\n"
		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("y\n"))
		_ = w.Close()

		oldStderr := os.Stderr
		_, stderrW, _ := os.Pipe()
		os.Stderr = stderrW

		approved, err := promptShellApproval()
		os.Stdin = oldStdin
		os.Stderr = oldStderr
		_ = stderrW.Close()

		require.NoError(t, err)
		assert.True(t, approved)
	})

	t.Run("Denied", func(t *testing.T) {
		// Enable test mode to use stdin instead of /dev/tty
		oldTestMode := os.Getenv("DIRVANA_TEST_MODE")
		require.NoError(t, os.Setenv("DIRVANA_TEST_MODE", "1"))
		defer func() {
			if oldTestMode == "" {
				_ = os.Unsetenv("DIRVANA_TEST_MODE")
			} else {
				_ = os.Setenv("DIRVANA_TEST_MODE", oldTestMode)
			}
		}()

		// Simulate user input "n\n"
		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("n\n"))
		_ = w.Close()

		oldStderr := os.Stderr
		_, stderrW, _ := os.Pipe()
		os.Stderr = stderrW

		approved, err := promptShellApproval()
		os.Stdin = oldStdin
		os.Stderr = oldStderr
		_ = stderrW.Close()

		require.NoError(t, err)
		assert.False(t, approved)
	})

	t.Run("YesFullWord", func(t *testing.T) {
		// Enable test mode to use stdin instead of /dev/tty
		oldTestMode := os.Getenv("DIRVANA_TEST_MODE")
		require.NoError(t, os.Setenv("DIRVANA_TEST_MODE", "1"))
		defer func() {
			if oldTestMode == "" {
				_ = os.Unsetenv("DIRVANA_TEST_MODE")
			} else {
				_ = os.Setenv("DIRVANA_TEST_MODE", oldTestMode)
			}
		}()

		// Simulate user input "yes\n"
		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("yes\n"))
		_ = w.Close()

		oldStderr := os.Stderr
		_, stderrW, _ := os.Pipe()
		os.Stderr = stderrW

		approved, err := promptShellApproval()
		os.Stdin = oldStdin
		os.Stderr = oldStderr
		_ = stderrW.Close()

		require.NoError(t, err)
		assert.True(t, approved)
	})
}

func TestAllowWithParams_AutoApproveShell_AllConfigFormats(t *testing.T) {
	// Regression: --auto-approve-shell used to hardcode .dirvana.yml and
	// silently no-op for the other supported config filenames.
	configs := map[string]string{
		".dirvana.yml": `env:
  CURRENT_USER:
    sh: whoami
`,
		".dirvana.yaml": `env:
  CURRENT_USER:
    sh: whoami
`,
		".dirvana.toml": `[env.CURRENT_USER]
sh = "whoami"
`,
		".dirvana.json": `{"env": {"CURRENT_USER": {"sh": "whoami"}}}`,
	}

	for filename, content := range configs {
		t.Run(filename, func(t *testing.T) {
			tmpDir := t.TempDir()
			authPath := filepath.Join(tmpDir, "auth.json")
			projectPath := filepath.Join(tmpDir, "project")
			require.NoError(t, os.MkdirAll(projectPath, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(projectPath, filename), []byte(content), 0o644))

			err := AllowWithParams(AllowParams{
				AuthPath:         authPath,
				PathToAllow:      projectPath,
				AutoApproveShell: true,
				LogLevel:         "error",
			})
			require.NoError(t, err)

			authMgr, err := auth.New(authPath)
			require.NoError(t, err)
			requiresApproval := authMgr.RequiresShellApproval(projectPath, map[string]string{"CURRENT_USER": "whoami"})
			assert.False(t, requiresApproval, "shell commands in %s should be auto-approved", filename)
		})
	}
}

func TestAllowWithParams_AutoApproveShell(t *testing.T) {
	t.Run("AutoApproveShellCommands", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))

		// Create a config file with shell commands
		configContent := `env:
  CURRENT_USER:
    sh: whoami
  CURRENT_DIR:
    sh: pwd
`
		configPath := filepath.Join(projectPath, ".dirvana.yml")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

		// Allow with auto-approve
		err := AllowWithParams(AllowParams{
			AuthPath:         authPath,
			PathToAllow:      projectPath,
			AutoApproveShell: true,
			LogLevel:         "error",
		})
		require.NoError(t, err)

		// Verify directory is allowed
		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(projectPath)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Verify shell commands are approved
		shellEnv := map[string]string{
			"CURRENT_USER": "whoami",
			"CURRENT_DIR":  "pwd",
		}
		requiresApproval := authMgr.RequiresShellApproval(projectPath, shellEnv)
		assert.False(t, requiresApproval, "Shell commands should be approved")
	})

	t.Run("AutoApproveWithoutShellCommands", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))

		// Create a config file without shell commands
		configContent := `env:
  STATIC_VAR: "value"
`
		configPath := filepath.Join(projectPath, ".dirvana.yml")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

		// Allow with auto-approve (should not fail even without shell commands)
		err := AllowWithParams(AllowParams{
			AuthPath:         authPath,
			PathToAllow:      projectPath,
			AutoApproveShell: true,
			LogLevel:         "error",
		})
		require.NoError(t, err)

		// Verify directory is allowed
		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(projectPath)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("AutoApproveWithoutConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))

		// No config file - nothing to approve, allow still succeeds
		err := AllowWithParams(AllowParams{
			AuthPath:         authPath,
			PathToAllow:      projectPath,
			AutoApproveShell: true,
			LogLevel:         "error",
		})
		require.NoError(t, err)

		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(projectPath)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("WithoutAutoApprove", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))

		// Create a config file with shell commands
		configContent := `env:
  TEST_VAR:
    sh: echo test
`
		configPath := filepath.Join(projectPath, ".dirvana.yml")
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))

		// allow now prompts for the sh: commands consent; decline it
		// deterministically through the test-mode stdin fallback
		t.Setenv("DIRVANA_TEST_MODE", "1")
		r, w, err := os.Pipe()
		require.NoError(t, err)
		origStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = origStdin }()
		_, err = w.WriteString("n\n")
		require.NoError(t, err)
		_ = w.Close()

		// Allow without auto-approve
		err = AllowWithParams(AllowParams{
			AuthPath:         authPath,
			PathToAllow:      projectPath,
			AutoApproveShell: false,
			LogLevel:         "error",
		})
		require.NoError(t, err)

		// Verify directory is allowed
		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(projectPath)
		require.NoError(t, err)
		assert.True(t, allowed)

		// Verify shell commands are NOT approved (consent was declined)
		shellEnv := map[string]string{
			"TEST_VAR": "echo test",
		}
		requiresApproval := authMgr.RequiresShellApproval(projectPath, shellEnv)
		assert.True(t, requiresApproval, "Shell commands should require approval")
	})

	t.Run("ShowTipWhenInAuthorizedDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))

		// Save original directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(originalDir)
			require.NoError(t, err)
		}()

		// Change to the project directory
		err = os.Chdir(projectPath)
		require.NoError(t, err)

		// Allow the current directory
		err = AllowWithParams(AllowParams{
			AuthPath:         authPath,
			PathToAllow:      projectPath,
			AutoApproveShell: false,
			LogLevel:         "error",
		})
		require.NoError(t, err)

		// The tip message should be printed, but we can't easily capture stdout
		// The important part is that the function completes without error
		// and the directory is allowed
		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(projectPath)
		require.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestHandleShellApproval(t *testing.T) {
	log := logger.New("error", os.Stderr)

	t.Run("AutoApproveCoversInheritedCommands", func(t *testing.T) {
		// Regression: --auto-approve-shell only approved the directory's
		// OWN sh: commands while the export gate checks the MERGED set,
		// so the flag silently failed in hierarchies (breaking CI usage)
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		parentDir := filepath.Join(tmpDir, "project")
		childDir := filepath.Join(parentDir, "child")
		require.NoError(t, os.MkdirAll(childDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(parentDir, ".dirvana.yml"),
			[]byte("env:\n  PARENT_USER:\n    sh: whoami\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(childDir, ".dirvana.yml"),
			[]byte("env:\n  CHILD_DIR:\n    sh: pwd\n"), 0o644))

		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		require.NoError(t, authMgr.Allow(parentDir))
		require.NoError(t, authMgr.Allow(childDir))

		require.NoError(t, handleShellApproval(childDir, authMgr, true, log))

		// The export gate checks the merged set: it must be satisfied
		mergedEnv := map[string]string{
			"PARENT_USER": "whoami",
			"CHILD_DIR":   "pwd",
		}
		assert.False(t, authMgr.RequiresShellApproval(childDir, mergedEnv),
			"auto-approve must cover inherited sh: commands")
	})

	t.Run("NoConfigAnywhere", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))

		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		require.NoError(t, authMgr.Allow(projectPath))

		// Nothing to approve, no error
		require.NoError(t, handleShellApproval(projectPath, authMgr, true, log))
	})

	t.Run("NoShellCommands", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".dirvana.yml"),
			[]byte("env:\n  STATIC: value\n"), 0o644))

		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		require.NoError(t, authMgr.Allow(projectPath))

		require.NoError(t, handleShellApproval(projectPath, authMgr, true, log))
	})

	t.Run("InvalidConfigFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".dirvana.yml"),
			[]byte("invalid: [yaml"), 0o644))

		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		require.NoError(t, authMgr.Allow(projectPath))

		err = handleShellApproval(projectPath, authMgr, true, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load config")
	})

	t.Run("InteractiveDeclineKeepsAllowValid", func(t *testing.T) {
		// Declining the sh: commands must not fail the allow: aliases stay
		// usable and the export gate will re-ask on the next cd
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".dirvana.yml"),
			[]byte("env:\n  USER:\n    sh: whoami\n"), 0o644))

		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		require.NoError(t, authMgr.Allow(projectPath))

		// Route the prompt to stdin and answer "n"
		t.Setenv("DIRVANA_TEST_MODE", "1")
		r, w, err := os.Pipe()
		require.NoError(t, err)
		origStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = origStdin }()
		_, err = w.WriteString("n\n")
		require.NoError(t, err)
		_ = w.Close()

		require.NoError(t, handleShellApproval(projectPath, authMgr, false, log))

		// Still unapproved: the gate must ask again
		assert.True(t, authMgr.RequiresShellApproval(projectPath, map[string]string{"USER": "whoami"}))
	})

	t.Run("InteractiveAccept", func(t *testing.T) {
		tmpDir := t.TempDir()
		authPath := filepath.Join(tmpDir, "auth.json")
		projectPath := filepath.Join(tmpDir, "project")
		require.NoError(t, os.MkdirAll(projectPath, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(projectPath, ".dirvana.yml"),
			[]byte("env:\n  USER:\n    sh: whoami\n"), 0o644))

		authMgr, err := auth.New(authPath)
		require.NoError(t, err)
		require.NoError(t, authMgr.Allow(projectPath))

		t.Setenv("DIRVANA_TEST_MODE", "1")
		r, w, err := os.Pipe()
		require.NoError(t, err)
		origStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = origStdin }()
		_, err = w.WriteString("y\n")
		require.NoError(t, err)
		_ = w.Close()

		require.NoError(t, handleShellApproval(projectPath, authMgr, false, log))
		assert.False(t, authMgr.RequiresShellApproval(projectPath, map[string]string{"USER": "whoami"}))
	})
}

func TestExport_DisabledViaEnv(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Set DIRVANA_ENABLED=false
	require.NoError(t, os.Setenv("DIRVANA_ENABLED", "false"))
	defer func() { _ = os.Unsetenv("DIRVANA_ENABLED") }()

	params := ExportParams{
		LogLevel:  "error",
		PrevDir:   "",
		CachePath: cachePath,
		AuthPath:  authPath,
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Export(params)
	require.NoError(t, err)

	_ = w.Close()
	os.Stdout = oldStdout

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should return empty output when disabled
	assert.Empty(t, output)
}

func TestCacheMergedConfig_WithLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create a config file
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
env:
  TEST_VAR: value
functions:
  test_func: |
    echo "test function"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	// Initialize components
	comps, err := initializeComponents(cachePath, authPath)
	require.NoError(t, err)

	// Load the config
	cfg, err := comps.config.Load(configPath)
	require.NoError(t, err)

	dctx := core.NewContext(cfg.GetAliases(), cfg.Functions)
	hierarchyHash := "test_hash"
	hierarchyPaths := []string{tmpDir}

	log := logger.New("error", os.Stderr)

	// Call cacheMergedConfig
	cacheMergedConfig(tmpDir, hierarchyHash, hierarchyPaths, cfg, dctx, comps, log)

	// Verify cache entry
	entry, found := comps.cache.Get(tmpDir)
	require.True(t, found, "Cache entry should exist")

	// Should have cleanup data (has local config)
	assert.NotNil(t, entry.Aliases, "Aliases should not be nil")
	assert.NotNil(t, entry.EnvVars, "EnvVars should not be nil")
	assert.NotNil(t, entry.Functions, "Functions should not be nil")
	assert.Contains(t, entry.Aliases, "test", "Should contain alias 'test'")
	assert.Contains(t, entry.EnvVars, "TEST_VAR", "Should contain env var 'TEST_VAR'")
	assert.Contains(t, entry.Functions, "test_func", "Should contain function 'test_func'")

	// Should have merged data (for performance)
	assert.NotNil(t, entry.MergedAliases, "MergedAliases should not be nil")
	assert.NotNil(t, entry.MergedFunctions, "MergedFunctions should not be nil")
	assert.Equal(t, hierarchyHash, entry.HierarchyHash, "HierarchyHash should match")
}

func TestCacheMergedConfig_WithoutLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Create a subdirectory WITHOUT a config file
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	// Create a config in the parent only
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	configContent := `aliases:
  test: echo test
env:
  TEST_VAR: value
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Authorize the directory
	err = Allow(authPath, tmpDir)
	require.NoError(t, err)

	// Initialize components
	comps, err := initializeComponents(cachePath, authPath)
	require.NoError(t, err)

	// Load the parent config (this is what would be loaded when in subDir)
	cfg, err := comps.config.Load(configPath)
	require.NoError(t, err)

	dctx := core.NewContext(cfg.GetAliases(), cfg.Functions)
	hierarchyHash := "test_hash"
	hierarchyPaths := []string{tmpDir}

	log := logger.New("error", os.Stderr)

	// Call cacheMergedConfig for the subdirectory (which has no local config)
	cacheMergedConfig(subDir, hierarchyHash, hierarchyPaths, cfg, dctx, comps, log)

	// Verify cache entry
	entry, found := comps.cache.Get(subDir)
	require.True(t, found, "Cache entry should exist")

	// Should NOT have cleanup data (no local config)
	assert.Nil(t, entry.Aliases, "Aliases should be nil (no local config)")
	assert.Nil(t, entry.EnvVars, "EnvVars should be nil (no local config)")
	assert.Nil(t, entry.Functions, "Functions should be nil (no local config)")

	// Should still have merged data (for performance)
	assert.NotNil(t, entry.MergedAliases, "MergedAliases should not be nil")
	assert.NotNil(t, entry.MergedFunctions, "MergedFunctions should not be nil")
	assert.Equal(t, hierarchyHash, entry.HierarchyHash, "HierarchyHash should match")
	assert.Equal(t, hierarchyPaths, entry.HierarchyPaths, "HierarchyPaths should match")
}

func TestCacheMergedConfig_EmptyHash(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")
	authPath := filepath.Join(tmpDir, "auth.json")

	// Initialize components
	comps, err := initializeComponents(cachePath, authPath)
	require.NoError(t, err)

	log := logger.New("error", os.Stderr)

	// Call with empty hierarchyHash - should not cache anything
	cacheMergedConfig(tmpDir, "", []string{}, nil, nil, comps, log)

	// Verify NO cache entry was created
	_, found := comps.cache.Get(tmpDir)
	assert.False(t, found, "No cache entry should exist when hierarchyHash is empty")
}

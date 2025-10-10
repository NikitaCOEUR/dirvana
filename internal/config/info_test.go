package config

import (
	"path/filepath"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetConfigDetails_NilConfig tests GetConfigDetails with nil config
func TestGetConfigDetails_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	details := GetConfigDetails(nil, authMgr, tmpDir)
	require.NotNil(t, details)

	assert.Empty(t, details.Aliases)
	assert.Empty(t, details.Functions)
	assert.Empty(t, details.EnvStatic)
	assert.Empty(t, details.EnvShell)
	assert.Empty(t, details.Flags)
}

// TestGetConfigDetails_WithFlags tests flag extraction
func TestGetConfigDetails_WithFlags(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Test with local_only flag
	cfg := &Config{
		LocalOnly: true,
	}
	details := GetConfigDetails(cfg, authMgr, tmpDir)
	assert.Contains(t, details.Flags, "local_only")
	assert.NotContains(t, details.Flags, "ignore_global")

	// Test with ignore_global flag
	cfg = &Config{
		IgnoreGlobal: true,
	}
	details = GetConfigDetails(cfg, authMgr, tmpDir)
	assert.Contains(t, details.Flags, "ignore_global")
	assert.NotContains(t, details.Flags, "local_only")

	// Test with both flags
	cfg = &Config{
		LocalOnly:    true,
		IgnoreGlobal: true,
	}
	details = GetConfigDetails(cfg, authMgr, tmpDir)
	assert.Contains(t, details.Flags, "local_only")
	assert.Contains(t, details.Flags, "ignore_global")
}

// TestGetConfigDetails_WithShellEnvApproved tests shell env with approval
func TestGetConfigDetails_WithShellEnvApproved(t *testing.T) {
	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Authorize and approve shell commands
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	shellEnv := map[string]string{"TEST_VAR": "echo test"}
	err = authMgr.ApproveShellCommands(tmpDir, shellEnv)
	require.NoError(t, err)

	cfg := &Config{
		Env: map[string]interface{}{
			"SHELL_VAR": map[string]interface{}{
				"sh": "echo test",
			},
		},
	}

	details := GetConfigDetails(cfg, authMgr, tmpDir)
	require.Contains(t, details.EnvShell, "SHELL_VAR")
	assert.True(t, details.EnvShell["SHELL_VAR"].Approved)
	assert.Equal(t, "echo test", details.EnvShell["SHELL_VAR"].Command)
}

// TestGetConfigDetails_WithNilAuthManager tests with nil auth manager
func TestGetConfigDetails_WithNilAuthManager(t *testing.T) {
	cfg := &Config{
		Env: map[string]interface{}{
			"SHELL_VAR": map[string]interface{}{
				"sh": "echo test",
			},
		},
	}

	details := GetConfigDetails(cfg, nil, "/test/dir")
	require.Contains(t, details.EnvShell, "SHELL_VAR")
	assert.False(t, details.EnvShell["SHELL_VAR"].Approved) // Should be false when authMgr is nil
}

// TestGetCompletionOverrides_NilConfig tests GetCompletionOverrides with nil config
func TestGetCompletionOverrides_NilConfig(t *testing.T) {
	result := GetCompletionOverrides(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

// TestGetCompletionOverrides_SimpleAliases tests with simple string aliases
func TestGetCompletionOverrides_SimpleAliases(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]interface{}{
			"simple": "echo test",
		},
	}

	result := GetCompletionOverrides(cfg)
	assert.Empty(t, result) // Simple aliases don't have completion overrides
}

// TestGetCompletionOverrides_WithCompletionDisabled tests alias with completion disabled
func TestGetCompletionOverrides_WithCompletionDisabled(t *testing.T) {
	cfg := &Config{
		Aliases: map[string]interface{}{
			"nocomp": map[string]interface{}{
				"command":    "echo test",
				"completion": false,
			},
		},
	}

	result := GetCompletionOverrides(cfg)
	assert.Empty(t, result) // completion: false doesn't create override
}

// TestConvertAliases_WithComplexAliases tests convertAliases with various alias types
func TestConvertAliases_WithComplexAliases(t *testing.T) {
	aliases := map[string]interface{}{
		"simple":        "echo simple",
		"complex_valid": map[string]interface{}{"command": "echo complex"},
		"complex_invalid": map[string]interface{}{
			"command": 123, // Invalid type
		},
		"no_command": map[string]interface{}{
			"other": "value",
		},
	}

	result := convertAliases(aliases)

	assert.Equal(t, "echo simple", result["simple"])
	assert.Equal(t, "echo complex", result["complex_valid"])
	assert.NotContains(t, result, "complex_invalid") // Should be skipped
	assert.NotContains(t, result, "no_command")      // Should be skipped
}

// TestGetFunctionsList tests getFunctionsList
func TestGetFunctionsList(t *testing.T) {
	functions := map[string]string{
		"func1": "echo 1",
		"func2": "echo 2",
		"func3": "echo 3",
	}

	result := getFunctionsList(functions)

	assert.Len(t, result, 3)
	assert.Contains(t, result, "func1")
	assert.Contains(t, result, "func2")
	assert.Contains(t, result, "func3")
}

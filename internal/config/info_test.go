package config

import (
	"os"
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

// TestGetHierarchyInfo_EmptyDirectory tests with no config files
func TestGetHierarchyInfo_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks for macOS compatibility
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Change to tmpDir so config search works
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	info, err := GetHierarchyInfo(tmpDir, authMgr)
	require.NoError(t, err)
	require.NotNil(t, info)

	// No configs should exist
	assert.Empty(t, info.LocalConfigs)
	// MergedConfig is not nil, but should be empty
	require.NotNil(t, info.MergedConfig)
	assert.Empty(t, info.MergedConfig.Aliases)
	assert.Empty(t, info.MergedConfig.Functions)
}

// TestGetHierarchyInfo_WithLocalConfig tests with a local config file
func TestGetHierarchyInfo_WithLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks for macOS compatibility
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Create and authorize the directory
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	// Create a local config
	configContent := `aliases:
  test: echo test
`
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to tmpDir
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	info, err := GetHierarchyInfo(tmpDir, authMgr)
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should have one local config
	require.Len(t, info.LocalConfigs, 1)
	assert.Equal(t, configPath, info.LocalConfigs[0].Path)
	assert.True(t, info.LocalConfigs[0].Authorized)
	assert.True(t, info.LocalConfigs[0].Loaded)
	assert.False(t, info.LocalConfigs[0].LocalOnly)

	// MergedConfig should exist
	require.NotNil(t, info.MergedConfig)
	assert.Len(t, info.MergedConfig.Aliases, 1)
}

// TestGetHierarchyInfo_WithUnauthorizedConfig tests with unauthorized config
func TestGetHierarchyInfo_WithUnauthorizedConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks for macOS compatibility
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Don't authorize - just create config
	configContent := `aliases:
  test: echo test
`
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to tmpDir
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	info, err := GetHierarchyInfo(tmpDir, authMgr)
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should have one local config
	require.Len(t, info.LocalConfigs, 1)
	assert.Equal(t, configPath, info.LocalConfigs[0].Path)
	assert.False(t, info.LocalConfigs[0].Authorized)
	assert.False(t, info.LocalConfigs[0].Loaded)
}

// TestGetHierarchyInfo_WithLocalOnly tests local_only flag
func TestGetHierarchyInfo_WithLocalOnly(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks for macOS compatibility
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Authorize
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	// Create config with local_only
	configContent := `aliases:
  test: echo test
local_only: true
`
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to tmpDir
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	info, err := GetHierarchyInfo(tmpDir, authMgr)
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should have local_only set
	require.Len(t, info.LocalConfigs, 1)
	assert.True(t, info.LocalConfigs[0].LocalOnly)
}

// TestGetHierarchyInfo_WithGlobalConfig tests with global configuration
func TestGetHierarchyInfo_WithGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks for macOS compatibility
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Create global config directory
	globalDir := filepath.Join(tmpDir, ".config", "dirvana")
	err = os.MkdirAll(globalDir, 0755)
	require.NoError(t, err)

	globalConfigPath := filepath.Join(globalDir, "global.yml")
	globalContent := `aliases:
  global_alias: echo global
`
	err = os.WriteFile(globalConfigPath, []byte(globalContent), 0644)
	require.NoError(t, err)

	// Set XDG_CONFIG_HOME to use our temp directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()
	err = os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	require.NoError(t, err)

	// Create local config
	configContent := `aliases:
  local_alias: echo local
`
	err = os.WriteFile(filepath.Join(tmpDir, ".dirvana.yml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Authorize
	err = authMgr.Allow(tmpDir)
	require.NoError(t, err)

	// Change to tmpDir
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	info, err := GetHierarchyInfo(tmpDir, authMgr)
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should have global config
	require.NotNil(t, info.GlobalConfig)
	assert.Equal(t, globalConfigPath, info.GlobalConfig.Path)
	assert.True(t, info.GlobalConfig.Exists)
	assert.True(t, info.GlobalConfig.Loaded)

	// Should have local config
	require.Len(t, info.LocalConfigs, 1)
}

// TestGetHierarchyInfo_WithMultipleConfigs tests hierarchical configs
func TestGetHierarchyInfo_WithMultipleConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks for macOS compatibility
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	authPath := filepath.Join(tmpDir, "auth.json")
	authMgr, err := auth.New(authPath)
	require.NoError(t, err)

	// Create parent config
	parentDir := filepath.Join(tmpDir, "parent")
	err = os.MkdirAll(parentDir, 0755)
	require.NoError(t, err)

	parentConfig := filepath.Join(parentDir, ".dirvana.yml")
	err = os.WriteFile(parentConfig, []byte("aliases:\n  parent: echo parent\n"), 0644)
	require.NoError(t, err)

	// Create child config
	childDir := filepath.Join(parentDir, "child")
	err = os.MkdirAll(childDir, 0755)
	require.NoError(t, err)

	childConfig := filepath.Join(childDir, ".dirvana.yml")
	err = os.WriteFile(childConfig, []byte("aliases:\n  child: echo child\n"), 0644)
	require.NoError(t, err)

	// Authorize both directories
	err = authMgr.Allow(parentDir)
	require.NoError(t, err)
	err = authMgr.Allow(childDir)
	require.NoError(t, err)

	// Change to child directory
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	err = os.Chdir(childDir)
	require.NoError(t, err)

	info, err := GetHierarchyInfo(childDir, authMgr)
	require.NoError(t, err)
	require.NotNil(t, info)

	// Should have both configs
	assert.Len(t, info.LocalConfigs, 2)

	// Both should be authorized and loaded
	for _, cfg := range info.LocalConfigs {
		assert.True(t, cfg.Authorized)
		assert.True(t, cfg.Loaded)
	}
}

// TestSummarizeWhen_NilCondition tests with nil when
func TestSummarizeWhen_NilCondition(t *testing.T) {
	result := summarizeWhen(nil)
	assert.Empty(t, result)
}

// TestSummarizeWhen_FileCondition tests file condition summary
func TestSummarizeWhen_FileCondition(t *testing.T) {
	when := &When{File: "package.json"}
	result := summarizeWhen(when)
	assert.Equal(t, "file:package.json", result)
}

// TestSummarizeWhen_VarCondition tests variable condition summary
func TestSummarizeWhen_VarCondition(t *testing.T) {
	when := &When{Var: "KUBECONFIG"}
	result := summarizeWhen(when)
	assert.Equal(t, "var:KUBECONFIG", result)
}

// TestSummarizeWhen_DirCondition tests directory condition summary
func TestSummarizeWhen_DirCondition(t *testing.T) {
	when := &When{Dir: "node_modules"}
	result := summarizeWhen(when)
	assert.Equal(t, "dir:node_modules", result)
}

// TestSummarizeWhen_CommandCondition tests command condition summary
func TestSummarizeWhen_CommandCondition(t *testing.T) {
	when := &When{Command: "docker"}
	result := summarizeWhen(when)
	assert.Equal(t, "cmd:docker", result)
}

// TestSummarizeWhen_MultipleAtomicConditions tests multiple atomic conditions (AND)
func TestSummarizeWhen_MultipleAtomicConditions(t *testing.T) {
	when := &When{
		File: "$KUBECONFIG",
		Var:  "KUBECONFIG",
	}
	result := summarizeWhen(when)
	// Should join with " + " for multiple atomics
	assert.Contains(t, result, "file:$KUBECONFIG")
	assert.Contains(t, result, "var:KUBECONFIG")
	assert.Contains(t, result, " + ")
}

// TestSummarizeWhen_AllCondition tests all (AND) composite condition
func TestSummarizeWhen_AllCondition(t *testing.T) {
	when := &When{
		All: []When{
			{Var: "AWS_PROFILE"},
			{Command: "aws"},
			{File: ".env"},
		},
	}
	result := summarizeWhen(when)
	assert.Contains(t, result, "all(")
	assert.Contains(t, result, "var:AWS_PROFILE")
	assert.Contains(t, result, "cmd:aws")
	assert.Contains(t, result, "file:.env")
}

// TestSummarizeWhen_AnyCondition tests any (OR) composite condition
func TestSummarizeWhen_AnyCondition(t *testing.T) {
	when := &When{
		Any: []When{
			{File: ".env.local"},
			{File: ".env"},
			{File: ".env.example"},
		},
	}
	result := summarizeWhen(when)
	assert.Contains(t, result, "any(")
	assert.Contains(t, result, "file:.env.local")
	assert.Contains(t, result, "file:.env")
	assert.Contains(t, result, " | ")
}

// TestSummarizeWhen_NestedConditions tests nested all/any conditions
func TestSummarizeWhen_NestedConditions(t *testing.T) {
	when := &When{
		All: []When{
			{Var: "AWS_PROFILE"},
			{
				Any: []When{
					{File: ".env.production"},
					{File: ".env"},
				},
			},
		},
	}
	result := summarizeWhen(when)
	assert.Contains(t, result, "all(")
	assert.Contains(t, result, "var:AWS_PROFILE")
	assert.Contains(t, result, "any(")
	assert.Contains(t, result, "file:.env.production")
	assert.Contains(t, result, "file:.env")
}

// TestSummarizeWhen_EmptyCondition tests empty when structure
func TestSummarizeWhen_EmptyCondition(t *testing.T) {
	when := &When{}
	result := summarizeWhen(when)
	assert.Empty(t, result)
}

// TestSummarizeWhen_EmptyAllCondition tests empty all array
func TestSummarizeWhen_EmptyAllCondition(t *testing.T) {
	when := &When{
		All: []When{},
	}
	result := summarizeWhen(when)
	// Empty all should result in empty string
	assert.Empty(t, result)
}

// TestConvertAliasesWithInfo_SimpleAliases tests simple aliases conversion
func TestConvertAliasesWithInfo_SimpleAliases(t *testing.T) {
	aliases := map[string]AliasConfig{
		"simple": {Command: "echo simple"},
		"test":   {Command: "npm test"},
	}

	result := convertAliasesWithInfo(aliases)

	assert.Len(t, result, 2)
	assert.Equal(t, "echo simple", result["simple"].Command)
	assert.False(t, result["simple"].HasWhen)
	assert.Empty(t, result["simple"].WhenSummary)
	assert.Empty(t, result["simple"].Else)

	assert.Equal(t, "npm test", result["test"].Command)
	assert.False(t, result["test"].HasWhen)
}

// TestConvertAliasesWithInfo_ConditionalAliases tests conditional aliases conversion
func TestConvertAliasesWithInfo_ConditionalAliases(t *testing.T) {
	aliases := map[string]AliasConfig{
		"k": {
			Command: "kubectl",
			When: &When{
				Var:  "KUBECONFIG",
				File: "$KUBECONFIG",
			},
			Else: "echo 'KUBECONFIG not set'",
		},
		"dev": {
			Command: "npm run dev",
			When: &When{
				File: "package.json",
			},
			Else: "echo 'package.json not found'",
		},
	}

	result := convertAliasesWithInfo(aliases)

	assert.Len(t, result, 2)

	// Check k alias
	assert.Equal(t, "kubectl", result["k"].Command)
	assert.True(t, result["k"].HasWhen)
	assert.Contains(t, result["k"].WhenSummary, "var:KUBECONFIG")
	assert.Contains(t, result["k"].WhenSummary, "file:$KUBECONFIG")
	assert.Equal(t, "echo 'KUBECONFIG not set'", result["k"].Else)

	// Check dev alias
	assert.Equal(t, "npm run dev", result["dev"].Command)
	assert.True(t, result["dev"].HasWhen)
	assert.Equal(t, "file:package.json", result["dev"].WhenSummary)
	assert.Equal(t, "echo 'package.json not found'", result["dev"].Else)
}

// TestConvertAliasesWithInfo_ConditionalWithoutElse tests conditional without else
func TestConvertAliasesWithInfo_ConditionalWithoutElse(t *testing.T) {
	aliases := map[string]AliasConfig{
		"test": {
			Command: "npm test",
			When: &When{
				Dir: "node_modules",
			},
		},
	}

	result := convertAliasesWithInfo(aliases)

	assert.Len(t, result, 1)
	assert.Equal(t, "npm test", result["test"].Command)
	assert.True(t, result["test"].HasWhen)
	assert.Equal(t, "dir:node_modules", result["test"].WhenSummary)
	assert.Empty(t, result["test"].Else)
}

// TestConvertAliasesWithInfo_MixedAliases tests mix of simple and conditional
func TestConvertAliasesWithInfo_MixedAliases(t *testing.T) {
	aliases := map[string]AliasConfig{
		"simple": {Command: "echo simple"},
		"conditional": {
			Command: "docker compose",
			When:    &When{Command: "docker"},
			Else:    "echo 'Docker not installed'",
		},
	}

	result := convertAliasesWithInfo(aliases)

	assert.Len(t, result, 2)

	// Simple alias
	assert.Equal(t, "echo simple", result["simple"].Command)
	assert.False(t, result["simple"].HasWhen)

	// Conditional alias
	assert.Equal(t, "docker compose", result["conditional"].Command)
	assert.True(t, result["conditional"].HasWhen)
	assert.Equal(t, "cmd:docker", result["conditional"].WhenSummary)
	assert.Equal(t, "echo 'Docker not installed'", result["conditional"].Else)
}

// TestConvertAliasesWithInfo_EmptyMap tests with empty alias map
func TestConvertAliasesWithInfo_EmptyMap(t *testing.T) {
	aliases := map[string]AliasConfig{}

	result := convertAliasesWithInfo(aliases)

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

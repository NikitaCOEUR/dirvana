package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	c := New()
	assert.NotNil(t, c)
}

func TestConfig_LoadYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")

	yamlContent := `
aliases:
  ll: ls -la
  gs: git status
functions:
  greet: |
    echo "Hello, $1!"
env:
  PROJECT_NAME: myproject
  LOG_LEVEL: debug
local_only: false
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	c := New()
	cfg, err := c.Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "ls -la", cfg.Aliases["ll"])
	assert.Equal(t, "git status", cfg.Aliases["gs"])
	assert.Contains(t, cfg.Functions["greet"], "Hello")
	assert.Equal(t, "myproject", cfg.Env["PROJECT_NAME"])
	assert.Equal(t, "debug", cfg.Env["LOG_LEVEL"])
	assert.False(t, cfg.LocalOnly)
}

func TestConfig_LoadTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.toml")

	tomlContent := `
local_only = true

[aliases]
ll = "ls -la"
gs = "git status"

[functions]
greet = "echo 'Hello, $1!'"

[env]
PROJECT_NAME = "myproject"
DEBUG = "true"
`
	require.NoError(t, os.WriteFile(configPath, []byte(tomlContent), 0644))

	c := New()
	cfg, err := c.Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "ls -la", cfg.Aliases["ll"])
	assert.Equal(t, "git status", cfg.Aliases["gs"])
	assert.Equal(t, "echo 'Hello, $1!'", cfg.Functions["greet"])
	assert.Equal(t, "myproject", cfg.Env["PROJECT_NAME"])
	assert.True(t, cfg.LocalOnly)
}

func TestConfig_LoadJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.json")

	jsonContent := `{
  "aliases": {
    "ll": "ls -la",
    "gs": "git status"
  },
  "functions": {
    "greet": "echo 'Hello, $1!'"
  },
  "env": {
    "PROJECT_NAME": "myproject",
    "DEBUG": "true"
  },
  "local_only": false
}`
	require.NoError(t, os.WriteFile(configPath, []byte(jsonContent), 0644))

	c := New()
	cfg, err := c.Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "ls -la", cfg.Aliases["ll"])
	assert.Equal(t, "git status", cfg.Aliases["gs"])
	assert.Equal(t, "myproject", cfg.Env["PROJECT_NAME"])
	assert.False(t, cfg.LocalOnly)
}

func TestConfig_LoadNonExistent(t *testing.T) {
	c := New()
	_, err := c.Load("/nonexistent/path/.dirvana.yml")
	assert.Error(t, err)
}

func TestConfig_Hash(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")

	content := `
aliases:
  ll: ls -la
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	c := New()
	hash1, err := c.Hash(configPath)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Same content should produce same hash
	hash2, err := c.Hash(configPath)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different content should produce different hash
	newContent := `
aliases:
  ll: ls -lah
`
	require.NoError(t, os.WriteFile(configPath, []byte(newContent), 0644))
	hash3, err := c.Hash(configPath)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func TestConfig_Merge(t *testing.T) {
	parent := &Config{
		Aliases:   map[string]interface{}{"ll": "ls -la", "gs": "git status"},
		Functions: map[string]string{"greet": "echo 'Hi'"},
		Env:       map[string]interface{}{"PARENT": "true", "SHARED": "parent"},
		LocalOnly: false,
	}

	child := &Config{
		Aliases:   map[string]interface{}{"gs": "git status --short", "gd": "git diff"},
		Functions: map[string]string{"bye": "echo 'Goodbye'"},
		Env:       map[string]interface{}{"CHILD": "true", "SHARED": "child"},
		LocalOnly: false,
	}

	merged := Merge(parent, child)

	// Child should override parent
	assert.Equal(t, "git status --short", merged.Aliases["gs"])
	assert.Equal(t, "git diff", merged.Aliases["gd"])
	assert.Equal(t, "ls -la", merged.Aliases["ll"])

	// Functions should be merged
	assert.Equal(t, "echo 'Hi'", merged.Functions["greet"])
	assert.Equal(t, "echo 'Goodbye'", merged.Functions["bye"])

	// Env should be merged with child overriding
	assert.Equal(t, "true", merged.Env["PARENT"])
	assert.Equal(t, "true", merged.Env["CHILD"])
	assert.Equal(t, "child", merged.Env["SHARED"])
}

func TestConfig_MergeWithLocalOnly(t *testing.T) {
	parent := &Config{
		Aliases: map[string]interface{}{"ll": "ls -la"},
	}

	child := &Config{
		Aliases:   map[string]interface{}{"gs": "git status"},
		LocalOnly: true,
	}

	// When child has local_only, parent should be ignored
	merged := Merge(parent, child)
	assert.Equal(t, "git status", merged.Aliases["gs"])
	assert.NotContains(t, merged.Aliases, "ll")
}

func TestFindConfigFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hierarchy: /root/.dirvana.yml -> /root/child/.dirvana.yml -> /root/child/grandchild
	rootDir := filepath.Join(tmpDir, "root")
	childDir := filepath.Join(rootDir, "child")
	grandchildDir := filepath.Join(childDir, "grandchild")

	require.NoError(t, os.MkdirAll(grandchildDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, ".dirvana.yml"), []byte("aliases:\n  root: echo root"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".dirvana.yml"), []byte("aliases:\n  child: echo child"), 0644))

	files, err := FindConfigFiles(grandchildDir)
	require.NoError(t, err)

	// Should find configs from child and root (in order from root to child)
	assert.Len(t, files, 2)
	assert.Contains(t, files[0], "root")
	assert.Contains(t, files[1], "child")
}

func TestFindConfigFiles_NoConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	files, err := FindConfigFiles(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestConfig_GetEnvVars_StaticOnly(t *testing.T) {
	cfg := &Config{
		Env: map[string]interface{}{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	staticVars, shellVars := cfg.GetEnvVars()
	assert.Len(t, staticVars, 2)
	assert.Len(t, shellVars, 0)
	assert.Equal(t, "value1", staticVars["VAR1"])
	assert.Equal(t, "value2", staticVars["VAR2"])
}

func TestConfig_GetEnvVars_ShellOnly(t *testing.T) {
	cfg := &Config{
		Env: map[string]interface{}{
			"GIT_BRANCH": map[string]interface{}{
				"sh": "git branch --show-current",
			},
			"CURRENT_DIR": map[string]interface{}{
				"sh": "pwd",
			},
		},
	}

	staticVars, shellVars := cfg.GetEnvVars()
	assert.Len(t, staticVars, 0)
	assert.Len(t, shellVars, 2)
	assert.Equal(t, "git branch --show-current", shellVars["GIT_BRANCH"])
	assert.Equal(t, "pwd", shellVars["CURRENT_DIR"])
}

func TestConfig_GetEnvVars_Mixed(t *testing.T) {
	cfg := &Config{
		Env: map[string]interface{}{
			"STATIC_VAR": "static_value",
			"GIT_BRANCH": map[string]interface{}{
				"sh": "git branch --show-current",
			},
			"PROJECT_NAME": "myproject",
		},
	}

	staticVars, shellVars := cfg.GetEnvVars()
	assert.Len(t, staticVars, 2)
	assert.Len(t, shellVars, 1)
	assert.Equal(t, "static_value", staticVars["STATIC_VAR"])
	assert.Equal(t, "myproject", staticVars["PROJECT_NAME"])
	assert.Equal(t, "git branch --show-current", shellVars["GIT_BRANCH"])
}

func TestConfig_LoadYAML_WithShellEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")

	yamlContent := `
aliases:
  gs: git status
env:
  PROJECT_NAME: myproject
  GIT_BRANCH:
    sh: git branch --show-current
  CURRENT_TIME:
    sh: date +%s
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	c := New()
	cfg, err := c.Load(configPath)
	require.NoError(t, err)

	staticVars, shellVars := cfg.GetEnvVars()
	assert.Equal(t, "myproject", staticVars["PROJECT_NAME"])
	assert.Equal(t, "git branch --show-current", shellVars["GIT_BRANCH"])
	assert.Equal(t, "date +%s", shellVars["CURRENT_TIME"])
}

func TestConfig_LoadHierarchy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create parent config
	parentDir := tmpDir
	parentConfig := filepath.Join(parentDir, ".dirvana.yml")
	parentContent := `
aliases:
  ll: ls -la
  gs: git status
env:
  PARENT_VAR: parent_value
`
	require.NoError(t, os.WriteFile(parentConfig, []byte(parentContent), 0644))

	// Create child dir and config
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))
	childConfig := filepath.Join(childDir, ".dirvana.yml")
	childContent := `
aliases:
  gd: git diff
env:
  CHILD_VAR: child_value
`
	require.NoError(t, os.WriteFile(childConfig, []byte(childContent), 0644))

	loader := New()
	merged, files, err := loader.LoadHierarchy(childDir)
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Should have both parent and child aliases
	assert.Len(t, merged.Aliases, 3)
	assert.Equal(t, "ls -la", merged.Aliases["ll"])
	assert.Equal(t, "git status", merged.Aliases["gs"])
	assert.Equal(t, "git diff", merged.Aliases["gd"])

	// Should have both parent and child env vars
	staticVars, _ := merged.GetEnvVars()
	assert.Equal(t, "parent_value", staticVars["PARENT_VAR"])
	assert.Equal(t, "child_value", staticVars["CHILD_VAR"])
}

func TestConfig_LoadHierarchy_LocalOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Create parent config
	parentDir := tmpDir
	parentConfig := filepath.Join(parentDir, ".dirvana.yml")
	parentContent := `
aliases:
  ll: ls -la
`
	require.NoError(t, os.WriteFile(parentConfig, []byte(parentContent), 0644))

	// Create child dir with local_only config
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))
	childConfig := filepath.Join(childDir, ".dirvana.yml")
	childContent := `
aliases:
  gd: git diff
local_only: true
`
	require.NoError(t, os.WriteFile(childConfig, []byte(childContent), 0644))

	loader := New()
	merged, files, err := loader.LoadHierarchy(childDir)
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Should only have child alias (local_only)
	assert.Len(t, merged.Aliases, 1)
	assert.Equal(t, "git diff", merged.Aliases["gd"])
	assert.True(t, merged.LocalOnly)
}

func TestConfig_LoadHierarchy_NoConfigs(t *testing.T) {
	tmpDir := t.TempDir()

	loader := New()
	merged, files, err := loader.LoadHierarchy(tmpDir)
	require.NoError(t, err)
	assert.Nil(t, files)
	assert.NotNil(t, merged)
	assert.Len(t, merged.Aliases, 0)
	assert.Len(t, merged.Functions, 0)
	assert.Len(t, merged.Env, 0)
}

func TestGetGlobalConfigPath(t *testing.T) {
	// Save original env
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Test with XDG_CONFIG_HOME set
	testXDG := "/tmp/test-config"
	_ = os.Setenv("XDG_CONFIG_HOME", testXDG)
	path, err := GetGlobalConfigPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(testXDG, "dirvana", GlobalConfigName), path)

	// Test without XDG_CONFIG_HOME (fallback to ~/.config)
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	path, err = GetGlobalConfigPath()
	require.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".config", "dirvana", GlobalConfigName), path)
}

func TestConfig_LoadHierarchy_WithGlobal(t *testing.T) {
	tmpDir := t.TempDir()

	// Save original env
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Set XDG_CONFIG_HOME to temp dir
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config
	globalDir := filepath.Join(tmpDir, "dirvana")
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	globalConfig := filepath.Join(globalDir, GlobalConfigName)
	globalContent := `aliases:
  g: git
  ll: ls -lah
env:
  GLOBAL_VAR: global_value
`
	require.NoError(t, os.WriteFile(globalConfig, []byte(globalContent), 0644))

	// Create a local config in a subdirectory
	localDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(localDir, 0755))
	localConfig := filepath.Join(localDir, ".dirvana.yml")
	localContent := `aliases:
  ll: ls -la  # Override global
  local: echo local
`
	require.NoError(t, os.WriteFile(localConfig, []byte(localContent), 0644))

	loader := New()
	merged, files, err := loader.LoadHierarchy(localDir)
	require.NoError(t, err)
	assert.Len(t, files, 2) // global + local

	// Check that both global and local are loaded
	assert.Equal(t, "git", merged.Aliases["g"])               // from global
	assert.Equal(t, "ls -la", merged.Aliases["ll"])           // overridden by local
	assert.Equal(t, "echo local", merged.Aliases["local"])    // from local
	assert.Equal(t, "global_value", merged.Env["GLOBAL_VAR"]) // from global
}

func TestConfig_LoadHierarchy_IgnoreGlobal(t *testing.T) {
	tmpDir := t.TempDir()

	// Save original env
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Set XDG_CONFIG_HOME to temp dir
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config
	globalDir := filepath.Join(tmpDir, "dirvana")
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	globalConfig := filepath.Join(globalDir, GlobalConfigName)
	globalContent := `aliases:
  g: git
  ll: ls -lah
`
	require.NoError(t, os.WriteFile(globalConfig, []byte(globalContent), 0644))

	// Create a local config with ignore_global in a subdirectory
	localDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(localDir, 0755))
	localConfig := filepath.Join(localDir, ".dirvana.yml")
	localContent := `ignore_global: true
aliases:
  local: echo local
`
	require.NoError(t, os.WriteFile(localConfig, []byte(localContent), 0644))

	loader := New()
	merged, files, err := loader.LoadHierarchy(localDir)
	require.NoError(t, err)
	assert.Len(t, files, 1) // only local, global was ignored

	// Should only have local alias
	assert.Len(t, merged.Aliases, 1)
	assert.Equal(t, "echo local", merged.Aliases["local"])
	assert.True(t, merged.IgnoreGlobal)
}

// MockAuthChecker implements AuthChecker for testing
type MockAuthChecker struct {
	authorizedPaths map[string]bool
}

func NewMockAuthChecker() *MockAuthChecker {
	return &MockAuthChecker{
		authorizedPaths: make(map[string]bool),
	}
}

func (m *MockAuthChecker) Allow(path string) {
	m.authorizedPaths[path] = true
}

func (m *MockAuthChecker) IsAllowed(path string) (bool, error) {
	return m.authorizedPaths[path], nil
}

func TestConfig_LoadHierarchyWithAuth_SkipsUnauthorized(t *testing.T) {
	// Create directory structure A/B/C
	tmpDir := t.TempDir()
	dirA := tmpDir // A (root)
	dirB := filepath.Join(dirA, "B")
	dirC := filepath.Join(dirB, "C")

	require.NoError(t, os.MkdirAll(dirC, 0755))

	// Create config files in each directory
	configA := filepath.Join(dirA, ".dirvana.yml")
	configB := filepath.Join(dirB, ".dirvana.yml")
	configC := filepath.Join(dirC, ".dirvana.yml")

	configContentA := `aliases:
  a_cmd: echo "from A"
env:
  A_VAR: "value_a"`

	configContentB := `aliases:
  b_cmd: echo "from B"
  dangerous: rm -rf /  # This should NOT be loaded
env:
  B_VAR: "value_b"`

	configContentC := `aliases:
  c_cmd: echo "from C"
env:
  C_VAR: "value_c"`

	require.NoError(t, os.WriteFile(configA, []byte(configContentA), 0644))
	require.NoError(t, os.WriteFile(configB, []byte(configContentB), 0644))
	require.NoError(t, os.WriteFile(configC, []byte(configContentC), 0644))

	// Create mock auth checker - authorize A and C but NOT B
	auth := NewMockAuthChecker()
	auth.Allow(dirA)
	auth.Allow(dirC)
	// Note: dirB is NOT authorized

	loader := New()

	// Load hierarchy from C with auth checks
	merged, files, err := loader.LoadHierarchyWithAuth(dirC, auth)
	require.NoError(t, err)

	// Should have loaded configs from A and C, but skipped B
	// files should only contain authorized config files
	expectedFiles := []string{configA, configC}
	assert.ElementsMatch(t, expectedFiles, files)

	// Verify merged config contains only A and C data
	aliases := merged.GetAliases()
	assert.Contains(t, aliases, "a_cmd")        // From A (authorized)
	assert.Contains(t, aliases, "c_cmd")        // From C (authorized)
	assert.NotContains(t, aliases, "b_cmd")     // From B (not authorized)
	assert.NotContains(t, aliases, "dangerous") // From B (not authorized)

	staticEnv, _ := merged.GetEnvVars()
	assert.Equal(t, "value_a", staticEnv["A_VAR"]) // From A
	assert.Equal(t, "value_c", staticEnv["C_VAR"]) // From C
	assert.NotContains(t, staticEnv, "B_VAR")      // From B (should be skipped)
}

func TestConfig_LoadHierarchyWithAuth_AllAuthorized(t *testing.T) {
	// Create directory structure A/B/C
	tmpDir := t.TempDir()
	dirA := tmpDir // A (root)
	dirB := filepath.Join(dirA, "B")
	dirC := filepath.Join(dirB, "C")

	require.NoError(t, os.MkdirAll(dirC, 0755))

	// Create config files
	configA := filepath.Join(dirA, ".dirvana.yml")
	configB := filepath.Join(dirB, ".dirvana.yml")
	configC := filepath.Join(dirC, ".dirvana.yml")

	configContentA := `aliases:
  a_cmd: echo "from A"`
	configContentB := `aliases:
  b_cmd: echo "from B"`
	configContentC := `aliases:
  c_cmd: echo "from C"`

	require.NoError(t, os.WriteFile(configA, []byte(configContentA), 0644))
	require.NoError(t, os.WriteFile(configB, []byte(configContentB), 0644))
	require.NoError(t, os.WriteFile(configC, []byte(configContentC), 0644))

	// Authorize all directories
	auth := NewMockAuthChecker()
	auth.Allow(dirA)
	auth.Allow(dirB)
	auth.Allow(dirC)

	loader := New()
	merged, files, err := loader.LoadHierarchyWithAuth(dirC, auth)
	require.NoError(t, err)

	// Should have loaded all configs
	expectedFiles := []string{configA, configB, configC}
	assert.ElementsMatch(t, expectedFiles, files)

	// All aliases should be present
	aliases := merged.GetAliases()
	assert.Contains(t, aliases, "a_cmd")
	assert.Contains(t, aliases, "b_cmd")
	assert.Contains(t, aliases, "c_cmd")
}

func TestConfig_LoadHierarchyWithAuth_NoAuthChecker(t *testing.T) {
	// When no auth checker is provided, should behave like original LoadHierarchy
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")

	configContent := `aliases:
  test: echo "test"`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	loader := New()

	// Both methods should return the same result when no auth checker is used
	merged1, files1, err1 := loader.LoadHierarchy(tmpDir)
	require.NoError(t, err1)

	merged2, files2, err2 := loader.LoadHierarchyWithAuth(tmpDir, nil)
	require.NoError(t, err2)

	assert.Equal(t, files1, files2)
	assert.Equal(t, merged1.Aliases, merged2.Aliases)
}

func TestHasLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// No config file
	assert.False(t, HasLocalConfig(tmpDir))

	// Create .dirvana.yml
	configPath := filepath.Join(tmpDir, ".dirvana.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("aliases:\n  test: echo test"), 0644))
	assert.True(t, HasLocalConfig(tmpDir))

	// Test with different config types
	_ = os.Remove(configPath)
	assert.False(t, HasLocalConfig(tmpDir))

	// .dirvana.yaml
	configPath = filepath.Join(tmpDir, ".dirvana.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("aliases:\n  test: echo test"), 0644))
	assert.True(t, HasLocalConfig(tmpDir))

	_ = os.Remove(configPath)

	// .dirvana.toml
	configPath = filepath.Join(tmpDir, ".dirvana.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("[aliases]\ntest = \"echo test\""), 0644))
	assert.True(t, HasLocalConfig(tmpDir))

	_ = os.Remove(configPath)

	// .dirvana.json
	configPath = filepath.Join(tmpDir, ".dirvana.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{\"aliases\":{\"test\":\"echo test\"}}"), 0644))
	assert.True(t, HasLocalConfig(tmpDir))
}

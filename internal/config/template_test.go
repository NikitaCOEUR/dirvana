package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandTemplate_DirvanaDir(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	result := cfg.expandTemplate("{{.DIRVANA_DIR}}")
	assert.Equal(t, "/tmp/test/project", result)
}

func TestExpandTemplate_UserWorkingDir(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)

	result := cfg.expandTemplate("{{.USER_WORKING_DIR}}")
	assert.Equal(t, cwd, result)
}

func TestExpandTemplate_CombinedVariables(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	result := cfg.expandTemplate("Project: {{.DIRVANA_DIR}}, CWD: {{.USER_WORKING_DIR}}")
	assert.Contains(t, result, "Project: /tmp/test/project")
	assert.Contains(t, result, "CWD: ")
}

func TestExpandTemplate_WithSprigFunctions(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "base function",
			template: "{{.DIRVANA_DIR | base}}",
			expected: "project",
		},
		{
			name:     "dir function",
			template: "{{.DIRVANA_DIR | dir}}",
			expected: "/tmp/test",
		},
		{
			name:     "upper function",
			template: "{{.DIRVANA_DIR | base | upper}}",
			expected: "PROJECT",
		},
		{
			name:     "sha256sum + trunc",
			template: "{{.DIRVANA_DIR | sha256sum | trunc 8}}",
			expected: "", // Just check it doesn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.expandTemplate(tt.template)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else {
				// Just verify it executed without error
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestExpandTemplate_InvalidTemplate(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	// Invalid template syntax should return original string
	result := cfg.expandTemplate("{{.INVALID")
	assert.Equal(t, "{{.INVALID", result)
}

func TestExpandTemplate_NoTemplate(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	// Plain text should pass through unchanged
	result := cfg.expandTemplate("plain text without templates")
	assert.Equal(t, "plain text without templates", result)
}

func TestExpandAliasVars_SimpleString(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Aliases: map[string]interface{}{
			"build": "cd {{.DIRVANA_DIR}} && make",
		},
	}

	err := cfg.expandAliasVars()
	require.NoError(t, err)

	assert.Equal(t, "cd /tmp/test/project && make", cfg.Aliases["build"])
}

func TestExpandAliasVars_WithCommand(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Aliases: map[string]interface{}{
			"deploy": map[string]interface{}{
				"command": "cd {{.DIRVANA_DIR}} && ./deploy.sh",
			},
		},
	}

	err := cfg.expandAliasVars()
	require.NoError(t, err)

	aliasMap := cfg.Aliases["deploy"].(map[string]interface{})
	assert.Equal(t, "cd /tmp/test/project && ./deploy.sh", aliasMap["command"])
}

func TestExpandAliasVars_WithElse(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Aliases: map[string]interface{}{
			"kubectl": map[string]interface{}{
				"command": "kubectl",
				"else":    "echo 'Config not found at {{.DIRVANA_DIR}}'",
			},
		},
	}

	err := cfg.expandAliasVars()
	require.NoError(t, err)

	aliasMap := cfg.Aliases["kubectl"].(map[string]interface{})
	assert.Equal(t, "echo 'Config not found at /tmp/test/project'", aliasMap["else"])
}

func TestExpandAliasVars_WithWhenConditions(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Aliases: map[string]interface{}{
			"test": map[string]interface{}{
				"command": "echo test",
				"when": map[string]interface{}{
					"file": "{{.DIRVANA_DIR}}/config.yml",
					"dir":  "{{.DIRVANA_DIR}}/build",
				},
			},
		},
	}

	err := cfg.expandAliasVars()
	require.NoError(t, err)

	aliasMap := cfg.Aliases["test"].(map[string]interface{})
	whenMap := aliasMap["when"].(map[string]interface{})
	assert.Equal(t, "/tmp/test/project/config.yml", whenMap["file"])
	assert.Equal(t, "/tmp/test/project/build", whenMap["dir"])
}

func TestExpandFunctionVars(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Functions: map[string]string{
			"goto":  "cd {{.DIRVANA_DIR}}",
			"build": "cd {{.DIRVANA_DIR}} && make build",
		},
	}

	err := cfg.expandFunctionVars()
	require.NoError(t, err)

	assert.Equal(t, "cd /tmp/test/project", cfg.Functions["goto"])
	assert.Equal(t, "cd /tmp/test/project && make build", cfg.Functions["build"])
}

func TestExpandEnvVars_SimpleString(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Env: map[string]interface{}{
			"PROJECT_ROOT": "{{.DIRVANA_DIR}}",
			"BUILD_DIR":    "{{.DIRVANA_DIR}}/build",
		},
	}

	err := cfg.expandEnvVars()
	require.NoError(t, err)

	assert.Equal(t, "/tmp/test/project", cfg.Env["PROJECT_ROOT"])
	assert.Equal(t, "/tmp/test/project/build", cfg.Env["BUILD_DIR"])
}

func TestExpandEnvVars_WithShellCommand(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Env: map[string]interface{}{
			"GIT_BRANCH": map[string]interface{}{
				"sh": "cd {{.DIRVANA_DIR}} && git branch --show-current",
			},
		},
	}

	err := cfg.expandEnvVars()
	require.NoError(t, err)

	envMap := cfg.Env["GIT_BRANCH"].(map[string]interface{})
	assert.Equal(t, "cd /tmp/test/project && git branch --show-current", envMap["sh"])
}

func TestExpandEnvVars_WithValue(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Env: map[string]interface{}{
			"CONFIG_PATH": map[string]interface{}{
				"value": "{{.DIRVANA_DIR}}/config.yml",
			},
		},
	}

	err := cfg.expandEnvVars()
	require.NoError(t, err)

	envMap := cfg.Env["CONFIG_PATH"].(map[string]interface{})
	assert.Equal(t, "/tmp/test/project/config.yml", envMap["value"])
}

func TestExpandWhenVars_FileCondition(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	when := map[string]interface{}{
		"file": "{{.DIRVANA_DIR}}/config.yml",
	}

	err := cfg.expandWhenVars(when)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/test/project/config.yml", when["file"])
}

func TestExpandWhenVars_DirCondition(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	when := map[string]interface{}{
		"dir": "{{.DIRVANA_DIR}}/build",
	}

	err := cfg.expandWhenVars(when)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/test/project/build", when["dir"])
}

func TestExpandWhenVars_AllCondition(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	when := map[string]interface{}{
		"all": []interface{}{
			map[string]interface{}{
				"file": "{{.DIRVANA_DIR}}/file1.txt",
			},
			map[string]interface{}{
				"dir": "{{.DIRVANA_DIR}}/dir1",
			},
		},
	}

	err := cfg.expandWhenVars(when)
	require.NoError(t, err)

	allConds := when["all"].([]interface{})
	fileCond := allConds[0].(map[string]interface{})
	dirCond := allConds[1].(map[string]interface{})

	assert.Equal(t, "/tmp/test/project/file1.txt", fileCond["file"])
	assert.Equal(t, "/tmp/test/project/dir1", dirCond["dir"])
}

func TestExpandWhenVars_AnyCondition(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	when := map[string]interface{}{
		"any": []interface{}{
			map[string]interface{}{
				"file": "{{.DIRVANA_DIR}}/file1.txt",
			},
			map[string]interface{}{
				"file": "{{.DIRVANA_DIR}}/file2.txt",
			},
		},
	}

	err := cfg.expandWhenVars(when)
	require.NoError(t, err)

	anyConds := when["any"].([]interface{})
	cond1 := anyConds[0].(map[string]interface{})
	cond2 := anyConds[1].(map[string]interface{})

	assert.Equal(t, "/tmp/test/project/file1.txt", cond1["file"])
	assert.Equal(t, "/tmp/test/project/file2.txt", cond2["file"])
}

func TestExpandVars_Integration(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Aliases: map[string]interface{}{
			"build": "cd {{.DIRVANA_DIR}} && make",
		},
		Functions: map[string]string{
			"goto": "cd {{.DIRVANA_DIR}}",
		},
		Env: map[string]interface{}{
			"PROJECT_ROOT": "{{.DIRVANA_DIR}}",
		},
	}

	err := cfg.ExpandVars()
	require.NoError(t, err)

	// Check all expansions happened
	assert.Equal(t, "cd /tmp/test/project && make", cfg.Aliases["build"])
	assert.Equal(t, "cd /tmp/test/project", cfg.Functions["goto"])
	assert.Equal(t, "/tmp/test/project", cfg.Env["PROJECT_ROOT"])
}

func TestLoad_WithTemplateExpansion(t *testing.T) {
	// Create a temporary directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".dirvana.yml")

	configContent := `
env:
  PROJECT_ROOT: "{{.DIRVANA_DIR}}"
  PROJECT_NAME: "{{.DIRVANA_DIR | base}}"

aliases:
  build: "cd {{.DIRVANA_DIR}} && make"

functions:
  goto: "cd {{.DIRVANA_DIR}}"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	loader := New()
	cfg, err := loader.Load(configPath)
	require.NoError(t, err)

	// Verify ConfigDir was set
	assert.Equal(t, tmpDir, cfg.ConfigDir)

	// Verify templates were expanded
	assert.Equal(t, tmpDir, cfg.Env["PROJECT_ROOT"])
	assert.Equal(t, filepath.Base(tmpDir), cfg.Env["PROJECT_NAME"])
	assert.Equal(t, "cd "+tmpDir+" && make", cfg.Aliases["build"])
	assert.Equal(t, "cd "+tmpDir, cfg.Functions["goto"])
}

func TestExpandTemplate_WithPathFunctions(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project/subfolder",
	}

	tests := []struct {
		name     string
		template string
		check    func(t *testing.T, result string)
	}{
		{
			name:     "clean function",
			template: "{{.DIRVANA_DIR | clean}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "/tmp/test/project/subfolder", filepath.Clean(result))
			},
		},
		{
			name:     "ext function on path with no extension",
			template: "{{.DIRVANA_DIR | ext}}",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "", result)
			},
		},
		{
			name:     "join paths",
			template: "{{.DIRVANA_DIR}}/build",
			check: func(t *testing.T, result string) {
				assert.Equal(t, "/tmp/test/project/subfolder/build", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.expandTemplate(tt.template)
			tt.check(t, result)
		})
	}
}

func TestExpandTemplate_MultipleNesting(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/home/user/projects/myapp",
	}

	// Test nested directory structure
	result := cfg.expandTemplate("{{.DIRVANA_DIR | dir | dir | base}}")
	assert.Equal(t, "user", result)
}

func TestExpandTemplate_ConditionalLogic(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	// Test with Sprig's conditional functions
	result := cfg.expandTemplate(`{{if .DIRVANA_DIR}}has dir{{else}}no dir{{end}}`)
	assert.Equal(t, "has dir", result)
}

func TestExpandEnvVars_EmptyMaps(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Env:       map[string]interface{}{},
	}

	err := cfg.expandEnvVars()
	require.NoError(t, err)
	assert.Empty(t, cfg.Env)
}

func TestExpandAliasVars_EmptyMaps(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Aliases:   map[string]interface{}{},
	}

	err := cfg.expandAliasVars()
	require.NoError(t, err)
	assert.Empty(t, cfg.Aliases)
}

func TestExpandFunctionVars_EmptyMaps(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
		Functions: map[string]string{},
	}

	err := cfg.expandFunctionVars()
	require.NoError(t, err)
	assert.Empty(t, cfg.Functions)
}

func TestExpandWhenVars_EmptyMap(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	when := map[string]interface{}{}
	err := cfg.expandWhenVars(when)
	require.NoError(t, err)
	assert.Empty(t, when)
}

func TestExpandTemplate_WithStringFunctions(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/tmp/test/project",
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "lower function",
			template: "{{.DIRVANA_DIR | base | lower}}",
			expected: "project",
		},
		{
			name:     "replace function",
			template: `{{.DIRVANA_DIR | base | replace "project" "app"}}`,
			expected: "app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.expandTemplate(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

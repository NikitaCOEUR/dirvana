package shell

import (
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestGenerator_Generate(t *testing.T) {
	g := NewGenerator()

	aliases := map[string]config.AliasConfig{
		"ll": {Command: "ls -la", Completion: nil},
		"gs": {Command: "git status", Completion: nil},
	}

	functions := map[string]string{
		"greet": "echo \"Hello, $1!\"",
		"bye":   "echo \"Goodbye!\"",
	}

	staticEnv := map[string]string{
		"PROJECT_NAME": "myproject",
		"DEBUG":        "true",
	}

	shellEnv := map[string]string{
		"GIT_BRANCH": "git rev-parse --abbrev-ref HEAD",
	}

	code := g.Generate(aliases, functions, staticEnv, shellEnv)

	// Check aliases are present
	assert.Contains(t, code, "alias ll='ls -la'")
	assert.Contains(t, code, "alias gs='git status'")

	// Check functions are present
	assert.Contains(t, code, "greet()")
	assert.Contains(t, code, "echo \"Hello, $1!\"")
	assert.Contains(t, code, "bye()")

	// Check static env vars are present
	assert.Contains(t, code, "export PROJECT_NAME='myproject'")
	assert.Contains(t, code, "export DEBUG='true'")

	// Check shell env vars are present
	assert.Contains(t, code, "export GIT_BRANCH=\"$(git rev-parse --abbrev-ref HEAD)\"")
}

func TestGenerator_GenerateEmpty(t *testing.T) {
	g := NewGenerator()

	code := g.Generate(nil, nil, nil, nil)

	// Should produce valid but empty shell code
	assert.NotEmpty(t, code)
}

func TestGenerator_GenerateAliasesOnly(t *testing.T) {
	g := NewGenerator()

	aliases := map[string]config.AliasConfig{
		"ll": {Command: "ls -la", Completion: nil},
	}

	code := g.Generate(aliases, nil, nil, nil)
	assert.Contains(t, code, "alias ll='ls -la'")
	assert.NotContains(t, code, "export")
}

func TestGenerator_GenerateFunctionsOnly(t *testing.T) {
	g := NewGenerator()

	functions := map[string]string{
		"greet": "echo \"Hello\"",
	}

	code := g.Generate(nil, functions, nil, nil)
	assert.Contains(t, code, "greet()")
	assert.Contains(t, code, "echo \"Hello\"")
}

func TestGenerator_GenerateEnvOnly(t *testing.T) {
	g := NewGenerator()

	staticEnv := map[string]string{
		"TEST": "value",
	}

	code := g.Generate(nil, nil, staticEnv, nil)
	assert.Contains(t, code, "export TEST='value'")
}

func TestGenerator_GenerateShellEnvOnly(t *testing.T) {
	g := NewGenerator()

	shellEnv := map[string]string{
		"CURRENT_DIR": "pwd",
		"GIT_BRANCH":  "git branch --show-current",
	}

	code := g.Generate(nil, nil, nil, shellEnv)
	assert.Contains(t, code, "export CURRENT_DIR=\"$(pwd)\"")
	assert.Contains(t, code, "export GIT_BRANCH=\"$(git branch --show-current)\"")
}

func TestGenerator_EscapeQuotes(t *testing.T) {
	g := NewGenerator()

	aliases := map[string]config.AliasConfig{
		"test": {Command: "echo 'hello'", Completion: nil},
	}

	code := g.Generate(aliases, nil, nil, nil)
	// Should properly escape quotes
	assert.Contains(t, code, "alias test=")
}

func TestGenerator_MultilineFunction(t *testing.T) {
	g := NewGenerator()

	functions := map[string]string{
		"complex": `if [ -z "$1" ]; then
  echo "No argument"
else
  echo "Argument: $1"
fi`,
	}

	code := g.Generate(nil, functions, nil, nil)
	assert.Contains(t, code, "complex()")
	assert.Contains(t, code, "if [ -z \"$1\" ]")
}

func TestGenerator_SpecialCharsInEnv(t *testing.T) {
	g := NewGenerator()

	staticEnv := map[string]string{
		"PATH_EXTRA": "/usr/local/bin:/usr/bin",
		"MESSAGE":    "Hello World!",
	}

	code := g.Generate(nil, nil, staticEnv, nil)
	assert.Contains(t, code, "export PATH_EXTRA='/usr/local/bin:/usr/bin'")
	assert.Contains(t, code, "export MESSAGE='Hello World!'")
}

func TestGenerator_EmptyValues(t *testing.T) {
	g := NewGenerator()

	aliases := map[string]config.AliasConfig{
		"empty": {Command: "", Completion: nil},
	}

	staticEnv := map[string]string{
		"EMPTY_VAR": "",
	}

	code := g.Generate(aliases, nil, staticEnv, nil)
	// Should handle empty values gracefully
	assert.Contains(t, code, "alias empty=")
	assert.Contains(t, code, "export EMPTY_VAR=")
}

func TestGenerator_CompletionInherit(t *testing.T) {
	g := NewGenerator()

	aliases := map[string]config.AliasConfig{
		"gp": {Command: "git push", Completion: "git"},
	}

	code := g.Generate(aliases, nil, nil, nil)
	assert.Contains(t, code, "alias gp='git push'")
	assert.Contains(t, code, "compdef gp=git")
}

func TestGenerator_CompletionDisabled(t *testing.T) {
	g := NewGenerator()

	aliases := map[string]config.AliasConfig{
		"noop": {Command: "echo nothing", Completion: false},
	}

	code := g.Generate(aliases, nil, nil, nil)
	assert.Contains(t, code, "alias noop='echo nothing'")
	// Should not contain completion commands
	assert.NotContains(t, code, "complete")
	assert.NotContains(t, code, "compdef")
}

func TestGenerator_CompletionCustom(t *testing.T) {
	g := NewGenerator()

	aliases := map[string]config.AliasConfig{
		"deploy": {
			Command: "./deploy.sh",
			Completion: config.CompletionConfig{
				Bash: "complete -W 'dev staging prod' deploy",
				Zsh:  "compdef '_arguments \"1: :(dev staging prod)\"' deploy",
			},
		},
	}

	code := g.Generate(aliases, nil, nil, nil)
	assert.Contains(t, code, "alias deploy='./deploy.sh'")
	assert.Contains(t, code, "complete -W 'dev staging prod' deploy")
	assert.Contains(t, code, "compdef '_arguments \"1: :(dev staging prod)\"' deploy")
}

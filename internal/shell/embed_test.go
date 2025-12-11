package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashTemplate_Embedded(t *testing.T) {
	// Test that bash template is embedded and not empty
	assert.NotEmpty(t, bashTemplate, "bash template should be embedded")
	assert.Contains(t, bashTemplate, "__dirvana_complete", "should contain completion function")
	assert.Contains(t, bashTemplate, "COMPREPLY", "should contain bash completion variable")
	assert.Contains(t, bashTemplate, "complete -o nosort", "should register completion")
}

func TestZshTemplate_Embedded(t *testing.T) {
	// Test that zsh template is embedded and not empty
	assert.NotEmpty(t, zshTemplate, "zsh template should be embedded")
	assert.Contains(t, zshTemplate, "compdef", "should register compdef")
}

func TestZshFunctionTemplate_Embedded(t *testing.T) {
	// Test that zsh function template is embedded and not empty
	assert.NotEmpty(t, zshFunctionTemplate, "zsh function template should be embedded")
	assert.Contains(t, zshFunctionTemplate, "__dirvana_complete_zsh", "should contain zsh completion function")
	assert.Contains(t, zshFunctionTemplate, "_describe", "should use _describe")
}

func TestBashTemplate_HasPlaceholder(t *testing.T) {
	// Test that template has placeholder for aliases
	assert.Contains(t, bashTemplate, "%s", "should have placeholder for aliases")

	// Test that placeholder is in the complete command
	assert.Contains(t, bashTemplate, "complete -o nosort -F __dirvana_complete %s", "complete command should have placeholder")
}

func TestZshTemplate_HasPlaceholder(t *testing.T) {
	// Test that template has placeholder for alias
	assert.Contains(t, zshTemplate, "%s", "should have placeholder for alias")

	// Test that placeholder is in the compdef command
	assert.Contains(t, zshTemplate, "compdef __dirvana_complete_zsh %s", "compdef command should have placeholder")
}

func TestBashTemplate_HasShebang(t *testing.T) {
	lines := strings.Split(bashTemplate, "\n")
	assert.True(t, strings.HasPrefix(lines[0], "#!/"), "should have shebang")
	assert.Contains(t, lines[0], "bash", "shebang should specify bash")
}

func TestZshFunctionTemplate_HasShebang(t *testing.T) {
	lines := strings.Split(zshFunctionTemplate, "\n")
	assert.True(t, strings.HasPrefix(lines[0], "#!/"), "should have shebang")
	assert.Contains(t, lines[0], "zsh", "shebang should specify zsh")
}

func TestBashTemplate_HasFallback(t *testing.T) {
	// Bash template should have file completion fallback
	assert.Contains(t, bashTemplate, "compgen -f", "should have file completion fallback")
	assert.Contains(t, bashTemplate, "compopt -o filenames", "should enable filenames option")
}

func TestZshFunctionTemplate_HasFallback(t *testing.T) {
	// Zsh function template should have file completion fallback
	assert.Contains(t, zshFunctionTemplate, "_files", "should have _files fallback")
}

func TestBashTemplate_FormatsDescriptions(t *testing.T) {
	// Bash template should format descriptions
	assert.Contains(t, bashTemplate, "__dirvana_format_descriptions", "should have format function")
	assert.Contains(t, bashTemplate, "longest", "should calculate longest completion")
	assert.Contains(t, bashTemplate, "COLUMNS", "should use terminal width")
}

func TestBashTemplate_HandlesDirectories(t *testing.T) {
	// Bash template should handle directories (no space after /)
	assert.Contains(t, bashTemplate, "case", "should use case statement")
	assert.Contains(t, bashTemplate, "*/", "should check for trailing slash")
	assert.Contains(t, bashTemplate, "compopt -o nospace", "should disable space")
}

func TestTemplates_NoHardcodedAliases(t *testing.T) {
	// Templates should not have hardcoded alias commands in actual code
	// (kubectl appears in comments as example, which is fine)

	// Check bash - should not have kubectl as actual alias/command
	assert.NotContains(t, bashTemplate, "alias kubectl=", "should not hardcode kubectl alias")
	assert.NotContains(t, bashTemplate, "alias docker=", "should not hardcode docker alias")

	// Check zsh - should not have kubectl as actual command
	assert.NotContains(t, zshTemplate, "compdef __dirvana_complete_zsh kubectl", "should not hardcode kubectl compdef")
	assert.NotContains(t, zshTemplate, "compdef __dirvana_complete_zsh docker", "should not hardcode docker compdef")

	// Should have placeholder instead
	assert.Contains(t, bashTemplate, "complete -o nosort -F __dirvana_complete %s", "should use placeholder")
	assert.Contains(t, zshTemplate, "compdef __dirvana_complete_zsh %s", "should use placeholder")
}

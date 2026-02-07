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
	assert.Contains(t, bashTemplate, "__dirvana_register_completion", "should contain register helper")
}

func TestZshFunctionTemplate_Embedded(t *testing.T) {
	// Test that zsh function template is embedded and not empty
	assert.NotEmpty(t, zshFunctionTemplate, "zsh function template should be embedded")
	assert.Contains(t, zshFunctionTemplate, "__dirvana_complete_zsh", "should contain zsh completion function")
	assert.Contains(t, zshFunctionTemplate, "_describe", "should use _describe")
	assert.Contains(t, zshFunctionTemplate, "__dirvana_register_completion", "should contain register helper")
}

func TestBashTemplate_HasRegisterHelper(t *testing.T) {
	// Test that template has the register helper instead of direct %s
	assert.Contains(t, bashTemplate, "__dirvana_register_completion()", "should have register function definition")
	assert.Contains(t, bashTemplate, "complete -o nosort -F __dirvana_complete", "should register dirvana completion")
	assert.NotContains(t, bashTemplate, "%s", "should not have old-style placeholder")
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

func TestFishFunctionTemplate_Embedded(t *testing.T) {
	assert.NotEmpty(t, fishFunctionTemplate, "fish function template should be embedded")
	assert.Contains(t, fishFunctionTemplate, "__dirvana_complete_fish", "should contain fish completion function")
	assert.Contains(t, fishFunctionTemplate, "underlying_cmd", "should have native detection logic")
}

func TestTemplates_NoHardcodedAliases(t *testing.T) {
	// Templates should not have hardcoded alias commands
	assert.NotContains(t, bashTemplate, "alias kubectl=", "should not hardcode kubectl alias")
	assert.NotContains(t, bashTemplate, "alias docker=", "should not hardcode docker alias")
}

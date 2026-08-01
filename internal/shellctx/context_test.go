//nolint:revive // Package name intentionally matches stdlib for internal consistency
package shellctx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCleanupCode(t *testing.T) {
	aliases := []string{"ll", "gs", "gd"}
	functions := []string{"mkcd", "greet"}
	envVars := []string{"PROJECT_NAME", "DEBUG", "GIT_BRANCH"}

	t.Run("bash", func(t *testing.T) {
		code := GenerateCleanupCode(aliases, functions, envVars, "bash")

		// Check aliases are unaliased
		assert.Contains(t, code, "unalias ll")
		assert.Contains(t, code, "unalias gs")
		assert.Contains(t, code, "unalias gd")

		// Note: We intentionally don't remove completions (complete -r / compdef -d)
		// because it's very slow. Once alias is removed, completion is harmless.
		assert.NotContains(t, code, "complete -r")
		assert.NotContains(t, code, "compdef -d")

		// Check functions are unset
		assert.Contains(t, code, "unset -f mkcd")
		assert.Contains(t, code, "unset -f greet")

		// Check env vars are unset
		assert.Contains(t, code, "unset PROJECT_NAME")
		assert.Contains(t, code, "unset DEBUG")
		assert.Contains(t, code, "unset GIT_BRANCH")

		// Should have error handling
		assert.Contains(t, code, "2>/dev/null || true")
	})

	t.Run("fish", func(t *testing.T) {
		code := GenerateCleanupCode(aliases, functions, envVars, "fish")

		// Check aliases are removed with functions -e
		assert.Contains(t, code, "functions -e ll")
		assert.Contains(t, code, "functions -e gs")
		assert.Contains(t, code, "functions -e gd")

		// Fish doesn't use unalias
		assert.NotContains(t, code, "unalias")

		// Check functions are removed with functions -e
		assert.Contains(t, code, "functions -e mkcd")
		assert.Contains(t, code, "functions -e greet")

		// Fish doesn't use unset -f
		assert.NotContains(t, code, "unset -f")

		// Check env vars are unset with set -e
		assert.Contains(t, code, "set -e PROJECT_NAME")
		assert.Contains(t, code, "set -e DEBUG")
		assert.Contains(t, code, "set -e GIT_BRANCH")

		// Should have fish error handling
		assert.Contains(t, code, "2>/dev/null; or true")
	})
}

func TestGenerateCleanupCode_Empty(t *testing.T) {
	code := GenerateCleanupCode(nil, nil, nil, "bash")

	// Should still have header
	assert.Contains(t, code, "# Dirvana cleanup")
}

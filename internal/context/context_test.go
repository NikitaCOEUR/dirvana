package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldCleanup(t *testing.T) {
	tests := []struct {
		name        string
		previousDir string
		currentDir  string
		want        bool
	}{
		{
			name:        "no previous context",
			previousDir: "",
			currentDir:  "/home/user/project",
			want:        false,
		},
		{
			name:        "same directory",
			previousDir: "/home/user/project",
			currentDir:  "/home/user/project",
			want:        false,
		},
		{
			name:        "entering subdirectory - keep context",
			previousDir: "/home/user/project",
			currentDir:  "/home/user/project/src",
			want:        false,
		},
		{
			name:        "entering deep subdirectory - keep context",
			previousDir: "/home/user/project",
			currentDir:  "/home/user/project/src/internal/pkg",
			want:        false,
		},
		{
			name:        "leaving to parent - cleanup",
			previousDir: "/home/user/project/src",
			currentDir:  "/home/user/project",
			want:        true,
		},
		{
			name:        "leaving to sibling - cleanup",
			previousDir: "/home/user/project1",
			currentDir:  "/home/user/project2",
			want:        true,
		},
		{
			name:        "leaving to completely different path - cleanup",
			previousDir: "/home/user/project",
			currentDir:  "/var/www/site",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldCleanup(tt.previousDir, tt.currentDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateCleanupCode(t *testing.T) {
	aliases := []string{"ll", "gs", "gd"}
	functions := []string{"mkcd", "greet"}
	envVars := []string{"PROJECT_NAME", "DEBUG", "GIT_BRANCH"}

	code := GenerateCleanupCode(aliases, functions, envVars)

	// Check aliases are unaliased
	assert.Contains(t, code, "unalias ll")
	assert.Contains(t, code, "unalias gs")
	assert.Contains(t, code, "unalias gd")

	// Check completions are removed for aliases
	assert.Contains(t, code, "complete -r ll")
	assert.Contains(t, code, "compdef -d ll")
	assert.Contains(t, code, "complete -r gs")
	assert.Contains(t, code, "compdef -d gs")

	// Check functions are unset
	assert.Contains(t, code, "unset -f mkcd")
	assert.Contains(t, code, "unset -f greet")

	// Check env vars are unset
	assert.Contains(t, code, "unset PROJECT_NAME")
	assert.Contains(t, code, "unset DEBUG")
	assert.Contains(t, code, "unset GIT_BRANCH")

	// Should have error handling
	assert.Contains(t, code, "2>/dev/null || true")
}

func TestGenerateCleanupCode_Empty(t *testing.T) {
	code := GenerateCleanupCode(nil, nil, nil)

	// Should still have header
	assert.Contains(t, code, "# Dirvana cleanup")
}

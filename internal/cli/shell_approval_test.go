package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The env sh: commands run automatically on every cd, so they need explicit
// consent. These tests cover how that consent is asked, recorded and reused.

func TestDisplayShellCommandsForApproval(t *testing.T) {
	t.Run("lists every command, sorted", func(t *testing.T) {
		// notifyUser falls back to stderr when no terminal is reachable
		out := captureStderr(t, func() {
			require.NoError(t, displayShellCommandsForApproval(map[string]string{
				"USER":       "whoami",
				"GIT_BRANCH": "git rev-parse --abbrev-ref HEAD",
			}))
		})

		assert.Contains(t, out, "This configuration contains dynamic shell commands")
		assert.Contains(t, out, "GIT_BRANCH: git rev-parse --abbrev-ref HEAD")
		assert.Contains(t, out, "USER: whoami")
		assert.Contains(t, out, "These commands will execute to set environment variables.")
		assert.Less(t, strings.Index(out, "GIT_BRANCH"), strings.Index(out, "USER"),
			"commands must be listed in a stable order")
	})

	t.Run("nothing to display", func(t *testing.T) {
		out := captureStderr(t, func() {
			require.NoError(t, displayShellCommandsForApproval(map[string]string{}))
		})
		assert.Empty(t, out)
	})
}

func TestPromptShellApproval(t *testing.T) {
	tests := []struct {
		answer   string
		approved bool
	}{
		{answer: "y", approved: true},
		{answer: "yes", approved: true},
		{answer: "Y", approved: true},
		{answer: "n", approved: false},
		{answer: "", approved: false},
		{answer: "whatever", approved: false},
	}

	for _, tt := range tests {
		t.Run("answer "+tt.answer, func(t *testing.T) {
			answerPrompt(t, tt.answer)

			var approved bool
			var err error
			out := captureStderr(t, func() {
				approved, err = promptShellApproval()
			})

			require.NoError(t, err)
			assert.Equal(t, tt.approved, approved)
			assert.Contains(t, out, "Approve execution?")
		})
	}
}

func TestPromptShellApproval_ClosedInput(t *testing.T) {
	// An input that ends without an answer is a refusal, not a crash
	answerPromptEOF(t)

	var approved bool
	var err error
	captureStderr(t, func() {
		approved, err = promptShellApproval()
	})

	require.Error(t, err)
	assert.False(t, approved)
}

func TestAllowWithParams_AutoApproveShell_AllConfigFormats(t *testing.T) {
	// Regression: --auto-approve-shell used to hardcode .dirvana.yml and
	// silently no-op for the other supported config filenames.
	configs := map[string]string{
		".dirvana.yml":  "env:\n  CURRENT_USER:\n    sh: whoami\n",
		".dirvana.yaml": "env:\n  CURRENT_USER:\n    sh: whoami\n",
		".dirvana.toml": "[env.CURRENT_USER]\nsh = \"whoami\"\n",
		".dirvana.json": `{"env": {"CURRENT_USER": {"sh": "whoami"}}}`,
	}

	for filename, content := range configs {
		t.Run(filename, func(t *testing.T) {
			env := newTestEnv(t)
			require.NoError(t, os.WriteFile(filepath.Join(env.Dir, filename), []byte(content), 0o644))

			out := captureStdout(t, func() {
				require.NoError(t, AllowWithParams(AllowParams{
					AuthPath:         env.AuthPath,
					PathToAllow:      env.Dir,
					AutoApproveShell: true,
					LogLevel:         "error",
				}))
			})
			assert.Contains(t, out, "Shell commands auto-approved")

			authMgr, err := auth.New(env.AuthPath)
			require.NoError(t, err)
			assert.False(t, authMgr.RequiresShellApproval(env.Dir, map[string]string{"CURRENT_USER": "whoami"}),
				"shell commands in %s should be auto-approved", filename)
		})
	}
}

func TestAllowWithParams_AutoApproveShell(t *testing.T) {
	t.Run("approves every command of the directory", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  CURRENT_USER:\n    sh: whoami\n  CURRENT_DIR:\n    sh: pwd\n")

		require.NoError(t, AllowWithParams(AllowParams{
			AuthPath:         env.AuthPath,
			PathToAllow:      env.Dir,
			AutoApproveShell: true,
			LogLevel:         "error",
		}))

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(env.Dir)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.False(t, authMgr.RequiresShellApproval(env.Dir, map[string]string{
			"CURRENT_USER": "whoami",
			"CURRENT_DIR":  "pwd",
		}))
	})

	t.Run("nothing to approve without sh commands", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  STATIC_VAR: \"value\"\n")

		out := captureStdout(t, func() {
			require.NoError(t, AllowWithParams(AllowParams{
				AuthPath:         env.AuthPath,
				PathToAllow:      env.Dir,
				AutoApproveShell: true,
				LogLevel:         "error",
			}))
		})
		assert.NotContains(t, out, "auto-approved")

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(env.Dir)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("nothing to approve without config", func(t *testing.T) {
		env := newTestEnv(t)

		require.NoError(t, AllowWithParams(AllowParams{
			AuthPath:         env.AuthPath,
			PathToAllow:      env.Dir,
			AutoApproveShell: true,
			LogLevel:         "error",
		}))

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(env.Dir)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("declining leaves the directory authorized", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  TEST_VAR:\n    sh: echo test\n")
		answerPrompt(t, "n")

		out := captureStdout(t, func() {
			require.NoError(t, AllowWithParams(AllowParams{
				AuthPath:    env.AuthPath,
				PathToAllow: env.Dir,
				LogLevel:    "error",
			}))
		})
		assert.Contains(t, out, "not approved")
		assert.Contains(t, out, "--auto-approve-shell")

		// Aliases and functions stay usable; only the sh: commands wait
		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		allowed, err := authMgr.IsAllowed(env.Dir)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.True(t, authMgr.RequiresShellApproval(env.Dir, map[string]string{"TEST_VAR": "echo test"}))
	})
}

func TestHandleShellApproval(t *testing.T) {
	t.Run("auto-approve covers inherited commands", func(t *testing.T) {
		// Regression: --auto-approve-shell only approved the directory's
		// OWN sh: commands while the export gate checks the MERGED set,
		// so the flag silently failed in hierarchies (breaking CI usage)
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  PARENT_USER:\n    sh: whoami\n")
		env.allow(t)

		child := filepath.Join(env.Dir, "child")
		env.writeConfigIn(t, child, "env:\n  CHILD_DIR:\n    sh: pwd\n")
		env.allowDir(t, child)

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		require.NoError(t, handleShellApproval(child, authMgr, true, testLogger()))

		assert.False(t, authMgr.RequiresShellApproval(child, map[string]string{
			"PARENT_USER": "whoami",
			"CHILD_DIR":   "pwd",
		}), "auto-approve must cover inherited sh: commands")
	})

	t.Run("no config anywhere", func(t *testing.T) {
		env := newTestEnv(t)
		env.allow(t)

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		require.NoError(t, handleShellApproval(env.Dir, authMgr, true, testLogger()))
	})

	t.Run("no shell commands", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  STATIC: value\n")
		env.allow(t)

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		require.NoError(t, handleShellApproval(env.Dir, authMgr, true, testLogger()))
	})

	t.Run("unloadable config", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "invalid: [yaml")
		env.allow(t)

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		err = handleShellApproval(env.Dir, authMgr, true, testLogger())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load config")
	})

	t.Run("interactive decline keeps the allow valid", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  USER:\n    sh: whoami\n")
		env.allow(t)
		answerPrompt(t, "n")

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		captureStdout(t, func() {
			require.NoError(t, handleShellApproval(env.Dir, authMgr, false, testLogger()))
		})

		// Still unapproved: the export gate must ask again
		assert.True(t, authMgr.RequiresShellApproval(env.Dir, map[string]string{"USER": "whoami"}))
	})

	t.Run("interactive accept", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  USER:\n    sh: whoami\n")
		env.allow(t)
		answerPrompt(t, "y")

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)
		out := captureStdout(t, func() {
			require.NoError(t, handleShellApproval(env.Dir, authMgr, false, testLogger()))
		})
		assert.Contains(t, out, "Shell commands approved")

		assert.False(t, authMgr.RequiresShellApproval(env.Dir, map[string]string{"USER": "whoami"}))
	})

	t.Run("already approved asks nothing", func(t *testing.T) {
		env := newTestEnv(t)
		env.writeConfig(t, "env:\n  CURRENT_USER:\n    sh: whoami\n")
		require.NoError(t, AllowWithParams(AllowParams{
			AuthPath:         env.AuthPath,
			PathToAllow:      env.Dir,
			AutoApproveShell: true,
			LogLevel:         "error",
		}))

		authMgr, err := auth.New(env.AuthPath)
		require.NoError(t, err)

		// The hash is unchanged: no consent to ask for again
		out := captureStdout(t, func() {
			require.NoError(t, handleShellApproval(env.Dir, authMgr, false, testLogger()))
		})
		assert.Empty(t, out)
	})
}

func TestMergedShellEnv_NoConfig(t *testing.T) {
	env := newTestEnv(t)

	authMgr, err := auth.New(env.AuthPath)
	require.NoError(t, err)

	shellEnv, err := mergedShellEnv(env.Dir, authMgr)
	require.NoError(t, err)
	assert.Empty(t, shellEnv)
}

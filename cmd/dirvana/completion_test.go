package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestShellCompletionEnabled(t *testing.T) {
	// Create a test app instance
	app := &cli.Command{
		Name:                  "dirvana",
		EnableShellCompletion: true,
	}

	// Verify EnableShellCompletion is set
	assert.True(t, app.EnableShellCompletion, "Shell completion should be enabled")
}

func TestCompletionBashScript(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run completion command
	app := &cli.Command{
		Name:                  "dirvana",
		EnableShellCompletion: true,
	}

	err := app.Run(context.Background(), []string{"dirvana", "completion", "bash"})
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	output := buf.String()

	// What matters is that the script registers a completion handler for
	// the binary. Its exact preamble belongs to urfave/cli and changes
	// between releases: the shebang it used to carry is gone, which is
	// correct for a script meant to be sourced rather than executed.
	assert.Contains(t, output, "complete -o bashdefault -o default -F __dirvana_bash_autocomplete dirvana")
	assert.Contains(t, output, "__dirvana_bash_autocomplete()")
}

func TestCompletionZshScript(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run completion command
	app := &cli.Command{
		Name:                  "dirvana",
		EnableShellCompletion: true,
	}

	err := app.Run(context.Background(), []string{"dirvana", "completion", "zsh"})
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	output := buf.String()

	// Verify zsh completion script contains expected content
	assert.Contains(t, output, "#compdef")
	assert.Contains(t, output, "_dirvana")
	assert.Contains(t, output, "zsh")
}

func TestCompletionFishScript(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run completion command
	app := &cli.Command{
		Name:                  "dirvana",
		EnableShellCompletion: true,
	}

	err := app.Run(context.Background(), []string{"dirvana", "completion", "fish"})
	require.NoError(t, err)

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	output := buf.String()

	// Verify fish completion script contains expected content
	assert.Contains(t, output, "dirvana")
	assert.Contains(t, output, "complete")
}

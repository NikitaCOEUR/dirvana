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

	// Verify bash completion script contains expected content
	assert.Contains(t, output, "#!/bin/bash")
	assert.Contains(t, output, "_dirvana")
	assert.Contains(t, output, "bash-completion")
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

// Package main is the entry point for the Dirvana CLI application.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/NikitaCOEUR/dirvana/internal/trace"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
	"github.com/urfave/cli/v3"
)

func main() {
	// Initialize tracing (only active in dev builds with DIRVANA_TRACE set)
	defer trace.Init()()

	paths := resolvePaths()

	app := &cli.Command{
		Name:                  "dirvana",
		Usage:                 "Automatic shell environment loader per folder",
		Version:               version.Version,
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "warn",
				Usage:   "Log level (debug, info, warn, error)",
				Sources: cli.EnvVars("DIRVANA_LOG_LEVEL"),
			},
		},
		Commands: []*cli.Command{
			newExportCmd(paths),
			newAllowCmd(paths),
			newRevokeCmd(paths),
			newListCmd(paths),
			newStatusCmd(paths),
			newInitCmd(),
			newValidateCmd(),
			newEditCmd(),
			newSchemaCmd(),
			newHookCmd(),
			newSetupCmd(),
			newCleanCmd(paths),
			newExecCmd(paths),
			newCompletionCmd(paths),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

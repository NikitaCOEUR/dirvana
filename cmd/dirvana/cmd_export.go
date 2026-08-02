package main

import (
	"context"

	dircli "github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/urfave/cli/v3"
)

func newExportCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export shell code for current folder",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "prev",
				Value:   "",
				Usage:   "Previous directory for context cleanup",
				Sources: cli.EnvVars("DIRVANA_PREV"),
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return dircli.Export(dircli.ExportParams{
				LogLevel:  cmd.String("log-level"),
				PrevDir:   cmd.String("prev"),
				CachePath: paths.cache,
				AuthPath:  paths.auth,
			})
		},
	}
}

package main

import (
	"context"

	dircli "github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/urfave/cli/v3"
)

func newStatusCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show current Dirvana configuration status",
		Action: func(_ context.Context, _ *cli.Command) error {
			return dircli.Status(dircli.StatusParams{
				CachePath: paths.cache,
				AuthPath:  paths.auth,
			})
		},
	}
}

func newInitCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create a sample project file in current folder or global config",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "global",
				Aliases: []string{"g"},
				Usage:   "Create global config file instead of local",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return dircli.Init(cmd.Bool("global"))
		},
	}
}

func newValidateCmd() *cli.Command {
	return &cli.Command{
		Name:      "validate",
		Usage:     "Validate a Dirvana configuration file",
		ArgsUsage: "[config-file]",
		Action: func(_ context.Context, cmd *cli.Command) error {
			configPath := ""
			if cmd.Args().Len() > 0 {
				configPath = cmd.Args().Get(0)
			}
			return dircli.Validate(configPath)
		},
	}
}

func newEditCmd() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit or create a Dirvana configuration file in current directory or global config",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "global",
				Aliases: []string{"g"},
				Usage:   "Edit global config file instead of local",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return dircli.Edit(cmd.Bool("global"))
		},
	}
}

func newSchemaCmd() *cli.Command {
	return &cli.Command{
		Name:      "schema",
		Usage:     "Display or export the JSON Schema for Dirvana configuration files",
		ArgsUsage: "[output-file]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (prints to stdout if not specified)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			outputPath := cmd.String("output")
			if outputPath == "" && cmd.Args().Len() > 0 {
				outputPath = cmd.Args().Get(0)
			}
			return dircli.Schema(outputPath)
		},
	}
}

func newCleanCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:  "clean",
		Usage: "Clean cache entries",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Clear all cache entries instead of just current directory hierarchy",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return dircli.Clean(dircli.CleanParams{
				CachePath: paths.cache,
				LogLevel:  cmd.String("log-level"),
				All:       cmd.Bool("all"),
			})
		},
	}
}

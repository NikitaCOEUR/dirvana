package main

import (
	"context"

	dircli "github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/urfave/cli/v3"
)

func newAllowCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:  "allow",
		Usage: "Authorize a project for automatic execution",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "auto-approve-shell",
				Usage: "Automatically approve shell commands in the config (useful for CI/CD)",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			pathToAllow, err := targetPathOrCwd(cmd.Args().Get(0))
			if err != nil {
				return err
			}

			return dircli.AllowWithParams(dircli.AllowParams{
				AuthPath:         paths.auth,
				PathToAllow:      pathToAllow,
				CachePath:        paths.cache,
				LogLevel:         cmd.String("log-level"),
				AutoApproveShell: cmd.Bool("auto-approve-shell"),
			})
		},
	}
}

func newRevokeCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:  "revoke",
		Usage: "Revoke authorization for a project",
		Action: func(_ context.Context, cmd *cli.Command) error {
			pathToRevoke, err := targetPathOrCwd(cmd.Args().Get(0))
			if err != nil {
				return err
			}

			return dircli.RevokeWithParams(dircli.RevokeParams{
				AuthPath:     paths.auth,
				PathToRevoke: pathToRevoke,
				CachePath:    paths.cache,
				LogLevel:     cmd.String("log-level"),
			})
		},
	}
}

func newListCmd(paths appPaths) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all authorized projects",
		Action: func(_ context.Context, _ *cli.Command) error {
			return dircli.List(paths.auth)
		},
	}
}

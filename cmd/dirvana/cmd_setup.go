package main

import (
	"context"
	"fmt"

	"github.com/NikitaCOEUR/dirvana/internal/setup"
	shellpkg "github.com/NikitaCOEUR/dirvana/internal/shell"
	"github.com/urfave/cli/v3"
)

func newHookCmd() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "Print shell hook code for manual installation",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "shell",
				Value:   "auto",
				Usage:   "Shell type: bash, zsh, or auto",
				Sources: cli.EnvVars("DIRVANA_SHELL"),
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			sh := shellpkg.Detect(cmd.String("shell"))
			hookCode, err := shellpkg.GenerateHookCode(sh, shellpkg.BinaryPath())
			if err != nil {
				return fmt.Errorf("failed to generate hook code: %w", err)
			}

			fmt.Println("# Add this to your shell config file:")
			fmt.Printf("# For %s: add to ~/.%src\n\n", sh, sh)
			fmt.Println(hookCode)

			return nil
		},
	}
}

func newSetupCmd() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Automatically install or uninstall shell hook",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "shell",
				Value:   "auto",
				Usage:   "Shell type: bash, zsh, or auto",
				Sources: cli.EnvVars("DIRVANA_SHELL"),
			},
			&cli.BoolFlag{
				Name:    "uninstall",
				Aliases: []string{"u"},
				Usage:   "Uninstall the shell hook instead of installing it",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			sh := shellpkg.Detect(cmd.String("shell"))

			var result *setup.Result
			var err error

			if cmd.Bool("uninstall") {
				result, err = setup.UninstallHook(sh)
			} else {
				result, err = setup.InstallHook(sh)
			}

			if err != nil {
				return err
			}

			fmt.Println(result.Message)
			if result.Updated && !cmd.Bool("uninstall") {
				fmt.Println("\nTo activate in current shell, run:")
				fmt.Printf("  source %s\n", result.RCFile)
			}

			return nil
		},
	}
}

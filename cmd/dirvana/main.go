// Package main is the entry point for the Dirvana CLI application.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	dircli "github.com/NikitaCOEUR/dirvana/internal/cli"
	"github.com/NikitaCOEUR/dirvana/internal/setup"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
	"github.com/urfave/cli/v3"
)

//nolint:gocyclo // Main function complexity is acceptable
func main() {
	// Get XDG paths
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, _ := os.UserHomeDir()
		cacheHome = filepath.Join(home, ".cache")
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}

	cachePath := filepath.Join(cacheHome, "dirvana", "cache.json")
	authPath := filepath.Join(dataHome, "dirvana", "authorized.json")

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
			{
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
						CachePath: cachePath,
						AuthPath:  authPath,
					})
				},
			},
			{
				Name:  "allow",
				Usage: "Authorize a project for automatic execution",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "auto-approve-shell",
						Usage: "Automatically approve shell commands in the config (useful for CI/CD)",
					},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					currentDir, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}

					pathToAllow := currentDir
					if cmd.Args().Len() > 0 {
						pathToAllow = cmd.Args().Get(0)
					}

					return dircli.AllowWithParams(dircli.AllowParams{
						AuthPath:         authPath,
						PathToAllow:      pathToAllow,
						CachePath:        cachePath,
						LogLevel:         cmd.String("log-level"),
						AutoApproveShell: cmd.Bool("auto-approve-shell"),
					})
				},
			},
			{
				Name:  "revoke",
				Usage: "Revoke authorization for a project",
				Action: func(_ context.Context, cmd *cli.Command) error {
					currentDir, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}

					pathToRevoke := currentDir
					if cmd.Args().Len() > 0 {
						pathToRevoke = cmd.Args().Get(0)
					}

					return dircli.RevokeWithParams(dircli.RevokeParams{
						AuthPath:     authPath,
						PathToRevoke: pathToRevoke,
						CachePath:    cachePath,
						LogLevel:     cmd.String("log-level"),
					})
				},
			},
			{
				Name:  "list",
				Usage: "List all authorized projects",
				Action: func(_ context.Context, _ *cli.Command) error {
					return dircli.List(authPath)
				},
			},
			{
				Name:  "status",
				Usage: "Show current Dirvana configuration status",
				Action: func(_ context.Context, _ *cli.Command) error {
					return dircli.Status(dircli.StatusParams{
						CachePath: cachePath,
						AuthPath:  authPath,
					})
				},
			},
			{
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
			},
			{
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
			},
			{
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
			},
			{
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
			},
			{
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
					shell := dircli.DetectShell(cmd.String("shell"))
					hookCode := dircli.GenerateHookCode(shell)

					fmt.Println("# Add this to your shell config file:")
					fmt.Printf("# For %s: add to ~/.%src\n\n", shell, shell)
					fmt.Println(hookCode)

					return nil
				},
			},
			{
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
					shell := dircli.DetectShell(cmd.String("shell"))

					var result *setup.Result
					var err error

					if cmd.Bool("uninstall") {
						result, err = setup.UninstallHook(shell)
					} else {
						result, err = setup.InstallHook(shell)
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
			},
			{
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
						CachePath: cachePath,
						LogLevel:  cmd.String("log-level"),
						All:       cmd.Bool("all"),
					})
				},
			},
			{
				Name:            "exec",
				Usage:           "Execute a dirvana-managed alias or function",
				ArgsUsage:       "<alias> [args...]",
				Hidden:          true, // Hidden from help - used internally by shell aliases
				SkipFlagParsing: true, // Don't parse flags - pass them directly to the wrapped command
				HideHelp:        true, // Don't show help for this internal command
				Action: func(_ context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() == 0 {
						return fmt.Errorf("alias name required")
					}

					alias := cmd.Args().Get(0)
					args := cmd.Args().Slice()[1:]

					return dircli.Exec(dircli.ExecParams{
						CachePath: cachePath,
						LogLevel:  cmd.String("log-level"),
						Alias:     alias,
						Args:      args,
					})
				},
			},
			{
				Name:            "completion",
				Usage:           "Generate shell completions for dirvana-managed aliases",
				ArgsUsage:       "[completion-args...]",
				Hidden:          true, // Hidden from help - used internally by completion functions
				SkipFlagParsing: true, // Don't parse flags - pass them directly to the wrapped command
				HideHelp:        true, // Don't show help for this internal command
				Action: func(_ context.Context, cmd *cli.Command) error {
					// Bash completion provides COMP_WORDS via args
					// and COMP_CWORD via DIRVANA_COMP_CWORD env var

					// IMPORTANT: Use os.Args directly instead of cmd.Args()
					// because urfave/cli treats "--" as a special separator
					// and filters it out, but we need it for kubectl completion
					var words []string
					foundCompletion := false
					skipFirstDoubleDash := true
					for _, arg := range os.Args {
						if arg == "completion" {
							foundCompletion = true
							continue
						}
						if foundCompletion {
							// Skip the first "--" which is just bash's separator
							// but keep subsequent "--" as they might be meaningful (e.g., kubectl -- ...)
							if arg == "--" && skipFirstDoubleDash {
								skipFirstDoubleDash = false
								continue
							}
							words = append(words, arg)
						}
					}

					// Get COMP_CWORD from environment
					cword := len(words) - 1 // default to last word
					if cwordStr := os.Getenv("DIRVANA_COMP_CWORD"); cwordStr != "" {
						_, _ = fmt.Sscanf(cwordStr, "%d", &cword) // Ignore errors, keep default
					}

					return dircli.Completion(dircli.CompletionParams{
						CachePath: cachePath,
						LogLevel:  cmd.String("log-level"),
						Words:     words,
						CWord:     cword,
					})
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

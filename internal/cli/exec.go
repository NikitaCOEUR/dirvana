package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/condition"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/derrors"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
)

// ExecParams contains parameters for the Exec command
type ExecParams struct {
	CachePath string
	AuthPath  string
	LogLevel  string
	Alias     string
	Args      []string
}

// Exec resolves and executes an alias or function defined by Dirvana
func Exec(params ExecParams) error {
	log := logger.New(params.LogLevel, os.Stderr)

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return derrors.NewExecutionError(params.Alias, "failed to get current directory", err)
	}

	// Get merged alias configs and functions from the full hierarchy
	aliases, functions, err := getMergedAliasConfigs(currentDir, params.CachePath, params.AuthPath)
	if err != nil {
		return derrors.NewConfigurationError(currentDir, "failed to load configuration", err)
	}

	if len(aliases) == 0 && len(functions) == 0 {
		return derrors.NewNotFoundError(params.Alias, fmt.Sprintf("no dirvana context found for alias '%s'", params.Alias))
	}

	// Resolve the command to execute
	command, err := resolveCommand(params, aliases, functions, currentDir, log)
	if err != nil {
		return err
	}

	// Execute the command via shell
	return executeCommand(params, command, log)
}

// resolveCommand resolves an alias or function and handles conditions/completion
func resolveCommand(params ExecParams, aliases map[string]config.AliasConfig, functions map[string]string, currentDir string, log *logger.Logger) (string, error) {
	// Check if alias exists
	aliasConf, foundAlias := aliases[params.Alias]
	functionBody, foundFunction := functions[params.Alias]

	if !foundAlias && !foundFunction {
		return "", derrors.NewNotFoundError(params.Alias, fmt.Sprintf("alias '%s' not found in dirvana context", params.Alias))
	}

	var command string

	if foundAlias {
		command = resolveAliasCommand(params, aliasConf, currentDir, log)
	} else {
		// Handle function
		command = "__dirvana_function__" + functionBody
		log.Debug().Str("function", params.Alias).Msg("Resolving function")
	}

	return command, nil
}

// resolveAliasCommand handles alias resolution with conditions and completion
func resolveAliasCommand(params ExecParams, aliasConf config.AliasConfig, currentDir string, log *logger.Logger) string {
	command := aliasConf.Command

	// Evaluate conditions if present
	if aliasConf.When != nil {
		log.Debug().Str("alias", params.Alias).Msg("Evaluating conditions")

		// Parse the When struct into a Condition
		cond, err := condition.Parse(aliasConf.When)
		if err != nil {
			// For now, return the main command if condition parsing fails
			// In the original code, this would return an error
			log.Debug().Err(err).Str("alias", params.Alias).Msg("Failed to parse conditions, using main command")
		} else {
			// Create evaluation context
			ctx := condition.Context{
				Env:        buildEnvMap(),
				WorkingDir: currentDir,
			}

			// Evaluate the condition
			ok, msg, err := cond.Evaluate(ctx)
			if err != nil {
				// For now, return the main command if evaluation fails
				log.Debug().Err(err).Str("alias", params.Alias).Msg("Failed to evaluate conditions, using main command")
			} else if !ok {
				// Condition not met
				if aliasConf.Else != "" {
					// Use fallback command
					log.Debug().
						Str("alias", params.Alias).
						Str("reason", msg).
						Msg("Condition not met, using fallback command")
					command = aliasConf.Else
				} else {
					// No fallback, would return error in original code
					log.Debug().
						Str("alias", params.Alias).
						Str("reason", msg).
						Msg("Condition not met, no fallback command")
				}
			} else {
				log.Debug().Str("alias", params.Alias).Msg("Conditions met")
			}
		}
	}

	// Check if this is a completion call
	if len(params.Args) > 0 && (params.Args[0] == "__complete" || params.Args[0] == "completion") {
		if aliasConf.Completion != nil {
			if s, ok := aliasConf.Completion.(string); ok && s != "" {
				command = s
				log.Debug().Str("alias", params.Alias).Str("completion_command", command).Msg("Using completion command for __complete or completion")
			}
		}
	}

	log.Debug().Str("alias", params.Alias).Str("command", command).Msg("Resolving alias")
	return command
}

// executeCommand executes the resolved command via shell
func executeCommand(params ExecParams, command string, log *logger.Logger) error {
	// Detect which shell to use
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash" // Fallback to bash (will be found via PATH)
	}

	// Find shell executable path
	execPath, err := exec.LookPath(shell)
	if err != nil {
		return derrors.NewExecutionError(params.Alias, fmt.Sprintf("shell not found: %s", shell), err)
	}

	// Detect shell type to use appropriate argument syntax
	shellType := parseShellFromPath(shell)
	if shellType == "" {
		// Fallback: try to detect from environment
		shellType = DetectShell("auto")
	}

	// Build argv for shell execution
	argv := buildShellArgs(shell, shellType, command, params.Args)

	log.Debug().
		Str("shell", shell).
		Str("argv", fmt.Sprintf("%q", argv)).
		Msg("Executing command via shell")

	// Execute the command via shell (replace current process)
	// This allows shell variable expansion, pipes, redirections, etc.
	err = syscall.Exec(execPath, argv, os.Environ())

	// If we reach here, syscall.Exec failed (extremely rare)
	return derrors.NewExecutionError(command, "failed to execute command", err)
}

// buildShellArgs builds the argument list for shell execution
func buildShellArgs(shell, shellType, command string, args []string) []string {
	// Bash/Zsh: shell -c 'command "$@"' shell args...
	// Fish: shell -c 'command $argv' args...
	var argv []string
	if len(args) > 0 {
		if shellType == ShellFish {
			// Fish uses $argv for positional arguments
			argv = []string{shell, "-c", command + " $argv"}
			argv = append(argv, args...)
		} else {
			// Bash/Zsh: Use "$@" for argument passing
			argv = []string{shell, "-c", command + ` "$@"`, shell}
			argv = append(argv, args...)
		}
	} else {
		// No user arguments, just execute the command
		argv = []string{shell, "-c", command}
	}
	return argv
}

// buildEnvMap creates a map of environment variables for condition evaluation
func buildEnvMap() map[string]string {
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		// Split on first '=' only
		for i := 0; i < len(env); i++ {
			if env[i] == '=' {
				key := env[:i]
				value := env[i+1:]
				envMap[key] = value
				break
			}
		}
	}
	return envMap
}

// findCacheEntry searches for a cache entry in the current directory or parent directories
func findCacheEntry(c *cache.Cache, dir string) (*cache.Entry, bool) {
	dir = filepath.Clean(dir)

	// Try current directory first
	if entry, found := c.Get(dir); found {
		return entry, true
	}

	// Walk up the directory tree
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent

		if entry, found := c.Get(dir); found {
			return entry, true
		}
	}

	return nil, false
}

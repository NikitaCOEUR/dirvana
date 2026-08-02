package cli

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/NikitaCOEUR/dirvana/internal/condition"
	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/core"
	"github.com/NikitaCOEUR/dirvana/internal/logger"
	"github.com/NikitaCOEUR/dirvana/internal/shell"
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
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get the merged context from the full hierarchy (cached)
	dctx, err := core.NewEngine(params.CachePath, params.AuthPath).Load(currentDir)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if dctx.Empty() {
		return fmt.Errorf("no dirvana context found for alias '%s'", params.Alias)
	}

	// Resolve the command to execute
	command, err := resolveCommand(params, dctx.Aliases, dctx.Functions, currentDir, log)
	if err != nil {
		return err
	}

	if command == "" {
		return fmt.Errorf("empty command for alias '%s'", params.Alias)
	}

	// Execute the command via shell
	return executeCommand(params, command, log)
}

// execve is a seam over syscall.Exec: the real call replaces the current
// process, which would silently kill the test binary (remaining tests AND
// the coverage flush) the moment a test reaches it
var execve = syscall.Exec

// resolveCommand resolves an alias or function and handles conditions/completion
func resolveCommand(params ExecParams, aliases map[string]config.AliasConfig, functions map[string]string, currentDir string, log *logger.Logger) (string, error) {
	// Check if alias exists
	aliasConf, foundAlias := aliases[params.Alias]
	functionBody, foundFunction := functions[params.Alias]

	if !foundAlias && !foundFunction {
		return "", fmt.Errorf("alias '%s' not found in dirvana context", params.Alias)
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
	// Detect shell type
	shellType := shell.Detect("auto")

	// Map shell type to executable name
	shellExec := shell.Executable(shellType)

	// Find shell executable path
	execPath, err := exec.LookPath(shellExec)
	if err != nil {
		return fmt.Errorf("shell not found: %s: %w", shellExec, err)
	}

	// Build argv for shell execution
	argv := shell.BuildArgs(shellExec, shellType, command, params.Args)

	log.Debug().
		Str("shell", shellExec).
		Str("argv", fmt.Sprintf("%q", argv)).
		Msg("Executing command via shell")

	// Execute the command via shell (replace current process)
	// This allows shell variable expansion, pipes, redirections, etc.
	err = execve(execPath, argv, os.Environ())

	// If we reach here, syscall.Exec failed (extremely rare)
	return fmt.Errorf("failed to execute command: %w", err)
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

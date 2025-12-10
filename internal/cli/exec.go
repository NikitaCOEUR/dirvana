package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/condition"
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
	// This respects global config, ignore_global, local_only, and authorization
	aliases, functions, err := getMergedAliasConfigs(currentDir, params.CachePath, params.AuthPath)
	if err != nil {
		return derrors.NewConfigurationError(currentDir, "failed to load configuration", err)
	}

	if len(aliases) == 0 && len(functions) == 0 {
		return derrors.NewNotFoundError(params.Alias, fmt.Sprintf("no dirvana context found for alias '%s'", params.Alias))
	}

	// Check if alias exists
	aliasConf, foundAlias := aliases[params.Alias]
	functionBody, foundFunction := functions[params.Alias]

	if !foundAlias && !foundFunction {
		return derrors.NewNotFoundError(params.Alias, fmt.Sprintf("alias '%s' not found in dirvana context", params.Alias))
	}

	var command string

	if foundAlias {
		// Handle alias with potential conditions
		command = aliasConf.Command

		// Evaluate conditions if present
		if aliasConf.When != nil {
			log.Debug().Str("alias", params.Alias).Msg("Evaluating conditions")

			// Parse the When struct into a Condition
			cond, err := condition.Parse(aliasConf.When)
			if err != nil {
				return derrors.NewConditionError(params.Alias, "failed to parse conditions", err)
			}

			// Create evaluation context
			ctx := condition.Context{
				Env:        buildEnvMap(),
				WorkingDir: currentDir,
			}

			// Evaluate the condition
			ok, msg, err := cond.Evaluate(ctx)
			if err != nil {
				return derrors.NewConditionError(params.Alias, "failed to evaluate conditions", err)
			}

			if !ok {
				// Condition not met
				if aliasConf.Else != "" {
					// Use fallback command
					log.Debug().
						Str("alias", params.Alias).
						Str("reason", msg).
						Msg("Condition not met, using fallback command")
					command = aliasConf.Else
				} else {
					// No fallback, return error
					return derrors.NewConditionError(params.Alias, fmt.Sprintf("condition not met:\n%s", msg), nil)
				}
			} else {
				log.Debug().Str("alias", params.Alias).Msg("Conditions met")
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
	} else {
		// Handle function
		command = "__dirvana_function__" + functionBody
		log.Debug().Str("function", params.Alias).Msg("Resolving function")
	}

	// Execute via shell to allow variable expansion, pipes, redirections, etc.
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
	// Bash/Zsh: shell -c 'command "$@"' shell args...
	// Fish: Different approach - fish -c doesn't support positional args the same way
	var argv []string
	if len(params.Args) > 0 {
		if shellType == ShellFish {
			// Fish doesn't support "$@" style argument passing with -c
			// We need to build the command inline or use a different approach
			// For now, use bash as a fallback for Fish when executing commands with args
			// This is a temporary workaround until we find a better solution
			bashPath, err := exec.LookPath("bash")
			if err == nil {
				// Use bash as execution shell, even if user shell is fish
				execPath = bashPath // Update execPath to bash
				argv = []string{bashPath, "-c", command + ` "$@"`, "bash"}
				argv = append(argv, params.Args...)
				log.Debug().Msg("Using bash for command execution (fish doesn't support arg passing with -c)")
			} else {
				// No bash available, construct command with quoted args
				// This is less safe but should work for simple cases
				quotedArgs := ""
				for _, arg := range params.Args {
					// Basic quoting - escape single quotes
					escaped := "'" + escapeForShell(arg) + "'"
					quotedArgs += " " + escaped
				}
				argv = []string{shell, "-c", command + quotedArgs}
			}
		} else {
			// Bash/Zsh: Use "$@" for argument passing
			argv = []string{shell, "-c", command + ` "$@"`, shell}
			argv = append(argv, params.Args...)
		}
	} else {
		// No user arguments, just execute the command
		argv = []string{shell, "-c", command}
	}

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

// escapeForShell escapes a string for safe use in shell commands
// This is a basic implementation - for production use, consider more robust escaping
func escapeForShell(s string) string {
	// Replace single quotes with '\''
	// This closes the quote, adds an escaped quote, then reopens the quote
	return strings.ReplaceAll(s, "'", `'\''`)
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

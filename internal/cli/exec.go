package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/NikitaCOEUR/dirvana/internal/cache"
	"github.com/NikitaCOEUR/dirvana/internal/condition"
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
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get merged alias configs and functions from the full hierarchy
	// This respects global config, ignore_global, local_only, and authorization
	aliases, functions, err := getMergedAliasConfigs(currentDir, params.CachePath, params.AuthPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(aliases) == 0 && len(functions) == 0 {
		return fmt.Errorf("no dirvana context found for alias '%s'", params.Alias)
	}

	// Check if alias exists
	aliasConf, foundAlias := aliases[params.Alias]
	functionBody, foundFunction := functions[params.Alias]

	if !foundAlias && !foundFunction {
		return fmt.Errorf("alias '%s' not found in dirvana context", params.Alias)
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
				return fmt.Errorf("failed to parse conditions for alias '%s': %w", params.Alias, err)
			}

			// Create evaluation context
			ctx := condition.Context{
				Env:        buildEnvMap(),
				WorkingDir: currentDir,
			}

			// Evaluate the condition
			ok, msg, err := cond.Evaluate(ctx)
			if err != nil {
				return fmt.Errorf("failed to evaluate conditions for alias '%s': %w", params.Alias, err)
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
					return fmt.Errorf("condition not met for alias '%s':\n%s", params.Alias, msg)
				}
			} else {
				log.Debug().Str("alias", params.Alias).Msg("Conditions met")
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
		return fmt.Errorf("shell not found: %s", shell)
	}

	// Build argv for shell execution
	// Use: shell -c 'command "$@"' shell args...
	// The first arg after the command becomes $0 (we use shell name)
	// The remaining args become $1, $2, $3, etc. which are captured by "$@"
	var argv []string
	if len(params.Args) > 0 {
		// Append "$@" to command to receive user arguments
		argv = []string{shell, "-c", command + ` "$@"`, shell}
		argv = append(argv, params.Args...)
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

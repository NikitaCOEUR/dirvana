// Package condition provides condition evaluation for alias execution
package condition

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Condition represents a testable condition
type Condition interface {
	// Evaluate tests the condition and returns:
	// - bool: true if condition is met, false otherwise
	// - string: user-friendly error message if condition fails
	// - error: technical error if evaluation failed (file system error, etc.)
	Evaluate(ctx Context) (bool, string, error)
}

// Context provides the environment for condition evaluation
type Context struct {
	// Env contains environment variables (key-value pairs)
	Env map[string]string
	// WorkingDir is the current working directory
	WorkingDir string
}

// expandEnv expands environment variables in a string using the context's env map
func (ctx Context) expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		if val, ok := ctx.Env[key]; ok {
			return val
		}
		return os.Getenv(key)
	})
}

// resolveRelativePath resolves a path relative to the working directory
// If the path is already absolute, returns it as-is
func (ctx Context) resolveRelativePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(ctx.WorkingDir, path)
}

// FileCondition tests if a file exists
type FileCondition struct {
	Path string // Path to file (supports env var expansion)
}

// Evaluate implements Condition
func (c FileCondition) Evaluate(ctx Context) (bool, string, error) {
	// Expand environment variables
	expandedPath := ctx.expandEnv(c.Path)

	// Resolve relative paths
	resolvedPath := ctx.resolveRelativePath(expandedPath)

	// Check if file exists
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf("file '%s' does not exist", c.Path), nil
		}
		return false, "", fmt.Errorf("failed to check file '%s': %w", c.Path, err)
	}

	// Make sure it's a file, not a directory
	if info.IsDir() {
		return false, fmt.Sprintf("'%s' is a directory, not a file", c.Path), nil
	}

	return true, "", nil
}

// VarCondition tests if an environment variable is set and non-empty
type VarCondition struct {
	Name string // Variable name
}

// Evaluate implements Condition
func (c VarCondition) Evaluate(ctx Context) (bool, string, error) {
	// Check in context env first
	if val, ok := ctx.Env[c.Name]; ok && val != "" {
		return true, "", nil
	}

	// Fallback to os env
	if val := os.Getenv(c.Name); val != "" {
		return true, "", nil
	}

	return false, fmt.Sprintf("environment variable '%s' is not set or empty", c.Name), nil
}

// DirCondition tests if a directory exists
type DirCondition struct {
	Path string // Path to directory (supports env var expansion)
}

// Evaluate implements Condition
func (c DirCondition) Evaluate(ctx Context) (bool, string, error) {
	// Expand environment variables
	expandedPath := ctx.expandEnv(c.Path)

	// Resolve relative paths
	resolvedPath := ctx.resolveRelativePath(expandedPath)

	// Check if directory exists
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf("directory '%s' does not exist", c.Path), nil
		}
		return false, "", fmt.Errorf("failed to check directory '%s': %w", c.Path, err)
	}

	// Make sure it's a directory, not a file
	if !info.IsDir() {
		return false, fmt.Sprintf("'%s' is a file, not a directory", c.Path), nil
	}

	return true, "", nil
}

// CommandCondition tests if a command exists in PATH
type CommandCondition struct {
	Name string // Command name
}

// Evaluate implements Condition
func (c CommandCondition) Evaluate(_ Context) (bool, string, error) {
	// Use exec.LookPath to search for command in PATH
	_, err := exec.LookPath(c.Name)
	if err != nil {
		return false, fmt.Sprintf("command '%s' not found in PATH", c.Name), nil
	}

	return true, "", nil
}

// AllCondition tests if all sub-conditions are true (AND logic)
type AllCondition struct {
	Conditions []Condition
}

// Evaluate implements Condition
func (c AllCondition) Evaluate(ctx Context) (bool, string, error) {
	var failedMessages []string

	for _, cond := range c.Conditions {
		ok, msg, err := cond.Evaluate(ctx)
		if err != nil {
			return false, "", err
		}
		if !ok {
			failedMessages = append(failedMessages, msg)
		}
	}

	if len(failedMessages) > 0 {
		// Return all failed conditions
		combinedMsg := ""
		for i, msg := range failedMessages {
			if i > 0 {
				combinedMsg += "\n  - "
			} else {
				combinedMsg += "  - "
			}
			combinedMsg += msg
		}
		return false, combinedMsg, nil
	}

	return true, "", nil
}

// AnyCondition tests if at least one sub-condition is true (OR logic)
type AnyCondition struct {
	Conditions []Condition
}

// Evaluate implements Condition
func (c AnyCondition) Evaluate(ctx Context) (bool, string, error) {
	var allMessages []string

	for _, cond := range c.Conditions {
		ok, msg, err := cond.Evaluate(ctx)
		if err != nil {
			return false, "", err
		}
		if ok {
			// At least one condition passed
			return true, "", nil
		}
		allMessages = append(allMessages, msg)
	}

	// All conditions failed
	combinedMsg := "none of the following conditions were met:\n"
	for _, msg := range allMessages {
		combinedMsg += "  - " + msg + "\n"
	}

	return false, combinedMsg, nil
}

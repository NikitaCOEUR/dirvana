package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthorizationError(t *testing.T) {
	cause := fmt.Errorf("permission denied")
	err := NewAuthorizationError("/test/path", "failed to authorize directory", cause)

	assert.Equal(t, "AUTH_ERROR", err.Code())
	assert.Equal(t, "/test/path", err.Path)
	assert.Contains(t, err.Error(), "failed to authorize directory")
	assert.Contains(t, err.Error(), "permission denied")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestConfigurationError(t *testing.T) {
	cause := fmt.Errorf("invalid YAML")
	err := NewConfigurationError("/path/to/config.yml", "failed to parse config", cause)

	assert.Equal(t, "CONFIG_ERROR", err.Code())
	assert.Equal(t, "/path/to/config.yml", err.Path)
	assert.Contains(t, err.Error(), "failed to parse config")
	assert.Contains(t, err.Error(), "invalid YAML")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestCacheError(t *testing.T) {
	cause := fmt.Errorf("file not found")
	err := NewCacheError("/cache/path", "failed to read cache", cause)

	assert.Equal(t, "CACHE_ERROR", err.Code())
	assert.Equal(t, "/cache/path", err.Path)
	assert.Contains(t, err.Error(), "failed to read cache")
	assert.Contains(t, err.Error(), "file not found")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestExecutionError(t *testing.T) {
	cause := fmt.Errorf("command not found")
	err := NewExecutionError("mycommand", "failed to execute", cause)

	assert.Equal(t, "EXEC_ERROR", err.Code())
	assert.Equal(t, "mycommand", err.Command)
	assert.Contains(t, err.Error(), "failed to execute")
	assert.Contains(t, err.Error(), "command not found")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestValidationError(t *testing.T) {
	cause := fmt.Errorf("invalid format")
	err := NewValidationError("email", "validation failed", cause)

	assert.Equal(t, "VALIDATION_ERROR", err.Code())
	assert.Equal(t, "email", err.Field)
	assert.Contains(t, err.Error(), "validation failed")
	assert.Contains(t, err.Error(), "invalid format")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("alias", "alias not found in context")

	assert.Equal(t, "NOT_FOUND", err.Code())
	assert.Equal(t, "alias", err.Resource)
	assert.Contains(t, err.Error(), "alias not found in context")
	assert.Nil(t, errors.Unwrap(err))
}

func TestAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("config.yml", "config file already exists")

	assert.Equal(t, "ALREADY_EXISTS", err.Code())
	assert.Equal(t, "config.yml", err.Resource)
	assert.Contains(t, err.Error(), "config file already exists")
	assert.Nil(t, errors.Unwrap(err))
}

func TestShellApprovalError(t *testing.T) {
	cause := fmt.Errorf("user rejected approval")
	err := NewShellApprovalError("/test/path", "shell commands not approved", cause)

	assert.Equal(t, "SHELL_APPROVAL_ERROR", err.Code())
	assert.Equal(t, "/test/path", err.Path)
	assert.Contains(t, err.Error(), "shell commands not approved")
	assert.Contains(t, err.Error(), "user rejected approval")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestConditionError(t *testing.T) {
	cause := fmt.Errorf("condition parsing failed")
	err := NewConditionError("myalias", "failed to evaluate condition", cause)

	assert.Equal(t, "CONDITION_ERROR", err.Code())
	assert.Equal(t, "myalias", err.Alias)
	assert.Contains(t, err.Error(), "failed to evaluate condition")
	assert.Contains(t, err.Error(), "condition parsing failed")
	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestErrorWithoutCause(t *testing.T) {
	err := NewAuthorizationError("/test/path", "simple error message", nil)

	assert.Equal(t, "AUTH_ERROR", err.Code())
	assert.Equal(t, "simple error message", err.Error())
	assert.Nil(t, errors.Unwrap(err))
}

func TestErrorChaining(t *testing.T) {
	rootCause := fmt.Errorf("root cause")
	configErr := NewConfigurationError("/config", "config error", rootCause)
	authErr := NewAuthorizationError("/path", "auth error", configErr)

	// Test unwrapping chain
	unwrapped := errors.Unwrap(authErr)
	assert.Equal(t, configErr, unwrapped)

	unwrapped = errors.Unwrap(unwrapped)
	assert.Equal(t, rootCause, unwrapped)
}

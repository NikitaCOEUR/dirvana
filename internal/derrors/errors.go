// Package derrors provides custom error types for Dirvana.
// These error types enable better error handling and more informative error messages
// throughout the application.
package derrors

import (
	"fmt"
)

// DirvanaError is the base interface for all Dirvana errors
type DirvanaError interface {
	error
	// Code returns a unique error code for programmatic error handling
	Code() string
}

// baseError provides common functionality for all Dirvana errors
type baseError struct {
	code    string
	message string
	cause   error
}

func (e *baseError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

func (e *baseError) Code() string {
	return e.code
}

func (e *baseError) Unwrap() error {
	return e.cause
}

// AuthorizationError represents errors related to directory authorization
type AuthorizationError struct {
	baseError
	Path string
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(path string, message string, cause error) *AuthorizationError {
	return &AuthorizationError{
		baseError: baseError{
			code:    "AUTH_ERROR",
			message: message,
			cause:   cause,
		},
		Path: path,
	}
}

// ConfigurationError represents errors in configuration files
type ConfigurationError struct {
	baseError
	Path string
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(path string, message string, cause error) *ConfigurationError {
	return &ConfigurationError{
		baseError: baseError{
			code:    "CONFIG_ERROR",
			message: message,
			cause:   cause,
		},
		Path: path,
	}
}

// CacheError represents errors in cache operations
type CacheError struct {
	baseError
	Path string
}

// NewCacheError creates a new cache error
func NewCacheError(path string, message string, cause error) *CacheError {
	return &CacheError{
		baseError: baseError{
			code:    "CACHE_ERROR",
			message: message,
			cause:   cause,
		},
		Path: path,
	}
}

// ExecutionError represents errors during command execution
type ExecutionError struct {
	baseError
	Command string
}

// NewExecutionError creates a new execution error
func NewExecutionError(command string, message string, cause error) *ExecutionError {
	return &ExecutionError{
		baseError: baseError{
			code:    "EXEC_ERROR",
			message: message,
			cause:   cause,
		},
		Command: command,
	}
}

// ValidationError represents errors during validation
type ValidationError struct {
	baseError
	Field string
}

// NewValidationError creates a new validation error
func NewValidationError(field string, message string, cause error) *ValidationError {
	return &ValidationError{
		baseError: baseError{
			code:    "VALIDATION_ERROR",
			message: message,
			cause:   cause,
		},
		Field: field,
	}
}

// NotFoundError represents errors when a resource is not found
type NotFoundError struct {
	baseError
	Resource string
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource string, message string) *NotFoundError {
	return &NotFoundError{
		baseError: baseError{
			code:    "NOT_FOUND",
			message: message,
			cause:   nil,
		},
		Resource: resource,
	}
}

// AlreadyExistsError represents errors when a resource already exists
type AlreadyExistsError struct {
	baseError
	Resource string
}

// NewAlreadyExistsError creates a new already exists error
func NewAlreadyExistsError(resource string, message string) *AlreadyExistsError {
	return &AlreadyExistsError{
		baseError: baseError{
			code:    "ALREADY_EXISTS",
			message: message,
			cause:   nil,
		},
		Resource: resource,
	}
}

// ShellApprovalError represents errors during shell command approval
type ShellApprovalError struct {
	baseError
	Path string
}

// NewShellApprovalError creates a new shell approval error
func NewShellApprovalError(path string, message string, cause error) *ShellApprovalError {
	return &ShellApprovalError{
		baseError: baseError{
			code:    "SHELL_APPROVAL_ERROR",
			message: message,
			cause:   cause,
		},
		Path: path,
	}
}

// ConditionError represents errors during condition evaluation
type ConditionError struct {
	baseError
	Alias string
}

// NewConditionError creates a new condition error
func NewConditionError(alias string, message string, cause error) *ConditionError {
	return &ConditionError{
		baseError: baseError{
			code:    "CONDITION_ERROR",
			message: message,
			cause:   cause,
		},
		Alias: alias,
	}
}

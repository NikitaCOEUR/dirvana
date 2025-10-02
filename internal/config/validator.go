package config

import (
	"fmt"
	"os"
	"strings"
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string
	Message string
}

// ValidationResult contains the results of config validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// Validate validates a config file
func Validate(path string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Try to load the config
	loader := New()
	cfg, err := loader.Load(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "syntax",
			Message: fmt.Sprintf("Failed to parse config: %v", err),
		})
		return result, nil
	}

	// Check for name conflicts between aliases and functions
	for aliasName := range cfg.Aliases {
		if _, exists := cfg.Functions[aliasName]; exists {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "aliases/" + aliasName,
				Message: fmt.Sprintf("Name conflict: '%s' is defined as both an alias and a function", aliasName),
			})
		}
	}

	// Validate environment variables
	for name, value := range cfg.Env {
		switch v := value.(type) {
		case string:
			// Static value - nothing special to validate
			continue
		case map[string]interface{}:
			// Shell-based env var
			if sh, ok := v["sh"].(string); ok {
				if strings.TrimSpace(sh) == "" {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   "env/" + name,
						Message: "Shell command is empty",
					})
				}
				// Basic validation: check for obviously invalid commands
				if strings.Contains(sh, "\n") {
					result.Valid = false
					result.Errors = append(result.Errors, ValidationError{
						Field:   "env/" + name,
						Message: "Shell command contains newlines (multiline commands not supported)",
					})
				}
			}
		}
	}

	// Validate aliases are not empty
	for name, value := range cfg.Aliases {
		var cmd string
		switch v := value.(type) {
		case string:
			cmd = v
		case map[string]interface{}:
			if c, ok := v["command"].(string); ok {
				cmd = c
			}
		}
		if strings.TrimSpace(cmd) == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "aliases/" + name,
				Message: "Alias command is empty",
			})
		}
	}

	// Validate functions are not empty
	for name, body := range cfg.Functions {
		if strings.TrimSpace(body) == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "functions/" + name,
				Message: "Function body is empty",
			})
		}
	}

	return result, nil
}

package config

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaJSON string

// GetSchemaJSON returns the JSON Schema for Dirvana configuration
func GetSchemaJSON() string {
	return schemaJSON
}

// ValidateWithSchema validates a config file against the JSON Schema
func ValidateWithSchema(path string, content []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Determine file format and convert to JSON-compatible structure
	var data interface{}

	switch {
	case len(path) > 4 && (path[len(path)-4:] == ".yml" || path[len(path)-5:] == ".yaml"):
		if err := yaml.Unmarshal(content, &data); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "syntax",
				Message: fmt.Sprintf("Invalid YAML syntax: %v", err),
			})
			return result, nil
		}
	case len(path) > 5 && path[len(path)-5:] == ".json":
		if err := json.Unmarshal(content, &data); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "syntax",
				Message: fmt.Sprintf("Invalid JSON syntax: %v", err),
			})
			return result, nil
		}
	case len(path) > 5 && path[len(path)-5:] == ".toml":
		// For TOML, use the existing loader
		loader := New()
		cfg, err := loader.Load(path)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "syntax",
				Message: fmt.Sprintf("Invalid TOML syntax: %v", err),
			})
			return result, nil
		}

		// Convert config to map
		data = map[string]interface{}{
			"aliases":       cfg.Aliases,
			"functions":     cfg.Functions,
			"env":           cfg.Env,
			"local_only":    cfg.LocalOnly,
			"ignore_global": cfg.IgnoreGlobal,
		}
	default:
		return nil, fmt.Errorf("unsupported file format")
	}

	// Load schema
	schemaLoader := gojsonschema.NewStringLoader(GetSchemaJSON())
	documentLoader := gojsonschema.NewGoLoader(data)

	// Validate
	validationResult, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %w", err)
	}

	if !validationResult.Valid() {
		result.Valid = false
		for _, err := range validationResult.Errors() {
			result.Errors = append(result.Errors, ValidationError{
				Field:   err.Field(),
				Message: err.Description(),
			})
		}
	}

	return result, nil
}

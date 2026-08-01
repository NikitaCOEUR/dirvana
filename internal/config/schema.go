package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/knadh/koanf/parsers/toml"
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

	// Determine file format and convert to JSON-compatible structure.
	// All formats are parsed into a generic map so the schema sees the
	// document as written, not a round-trip through typed config structs.
	var data interface{}

	switch filepath.Ext(path) {
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(content, &data); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "syntax",
				Message: fmt.Sprintf("Invalid YAML syntax: %v", err),
			})
			return result, nil
		}
	case ".json":
		if err := json.Unmarshal(content, &data); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "syntax",
				Message: fmt.Sprintf("Invalid JSON syntax: %v", err),
			})
			return result, nil
		}
	case ".toml":
		parsed, err := toml.Parser().Unmarshal(content)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "syntax",
				Message: fmt.Sprintf("Invalid TOML syntax: %v", err),
			})
			return result, nil
		}
		data = parsed
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

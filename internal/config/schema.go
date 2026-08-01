package config

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaJSON string

// GetSchemaJSON returns the JSON Schema for Dirvana configuration
func GetSchemaJSON() string {
	return schemaJSON
}

// compiledSchema compiles the embedded schema once
var compiledSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("dirvana.schema.json", doc); err != nil {
		return nil, fmt.Errorf("failed to load embedded schema: %w", err)
	}
	return compiler.Compile("dirvana.schema.json")
})

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

	schema, err := compiledSchema()
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %w", err)
	}

	// Round-trip through JSON so YAML/TOML values (e.g. int vs float64)
	// take the shapes the validator expects
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize document: %w", err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to normalize document: %w", err)
	}

	if err := schema.Validate(instance); err != nil {
		var verr *jsonschema.ValidationError
		if !errors.As(err, &verr) {
			return nil, fmt.Errorf("schema validation error: %w", err)
		}
		result.Valid = false
		appendValidationErrors(verr, result)
	}

	return result, nil
}

// errorPrinter renders schema validation messages
var errorPrinter = message.NewPrinter(language.English)

// appendValidationErrors flattens the validation error tree into
// ValidationErrors, keeping only the leaf causes (the actual violations)
func appendValidationErrors(verr *jsonschema.ValidationError, result *ValidationResult) {
	if len(verr.Causes) == 0 {
		field := "/" + strings.Join(verr.InstanceLocation, "/")
		result.Errors = append(result.Errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s: %s", field, verr.ErrorKind.LocalizedString(errorPrinter)),
		})
		return
	}
	for _, cause := range verr.Causes {
		appendValidationErrors(cause, result)
	}
}

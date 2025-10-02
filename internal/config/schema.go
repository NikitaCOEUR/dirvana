package config

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// GetSchemaJSON returns the JSON Schema for Dirvana configuration
func GetSchemaJSON() string {
	return `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json",
  "title": "Dirvana Configuration",
  "description": "Configuration file for Dirvana - automatic shell environment loader per folder",
  "type": "object",
  "properties": {
    "aliases": {
      "type": "object",
      "description": "Shell aliases - shortcuts for common commands",
      "patternProperties": {
        "^[a-zA-Z_][a-zA-Z0-9_-]*$": {
          "oneOf": [
            {
              "type": "string",
              "minLength": 1,
              "description": "Simple alias: command to execute (auto-detects completion)"
            },
            {
              "type": "object",
              "description": "Advanced alias with completion control",
              "properties": {
                "command": {
                  "type": "string",
                  "minLength": 1,
                  "description": "Command to execute"
                },
                "completion": {
                  "oneOf": [
                    {
                      "type": "string",
                      "minLength": 1,
                      "description": "Inherit completion from another command (e.g., 'git')"
                    },
                    {
                      "type": "boolean",
                      "description": "Set to false to disable completion"
                    },
                    {
                      "type": "object",
                      "description": "Custom shell completion code",
                      "properties": {
                        "bash": {
                          "type": "string",
                          "description": "Bash completion code"
                        },
                        "zsh": {
                          "type": "string",
                          "description": "Zsh completion code"
                        }
                      },
                      "additionalProperties": false
                    }
                  ],
                  "description": "Completion configuration (optional)"
                }
              },
              "required": ["command"],
              "additionalProperties": false
            }
          ]
        }
      },
      "additionalProperties": false
    },
    "functions": {
      "type": "object",
      "description": "Shell functions - reusable command sequences",
      "patternProperties": {
        "^[a-zA-Z_][a-zA-Z0-9_-]*$": {
          "type": "string",
          "minLength": 1,
          "description": "Function body (shell script)"
        }
      },
      "additionalProperties": false
    },
    "env": {
      "type": "object",
      "description": "Environment variables (static or dynamic via shell commands)",
      "patternProperties": {
        "^[a-zA-Z_][a-zA-Z0-9_]*$": {
          "oneOf": [
            {
              "type": "string",
              "description": "Static environment variable value"
            },
            {
              "type": "object",
              "description": "Dynamic environment variable (executed via shell)",
              "properties": {
                "sh": {
                  "type": "string",
                  "minLength": 1,
                  "pattern": "^[^\n]+$",
                  "description": "Shell command to execute (output becomes env var value)"
                },
                "value": {
                  "type": "string",
                  "description": "Alternative: static value (use string directly instead)"
                }
              },
              "oneOf": [
                {
                  "required": ["sh"]
                },
                {
                  "required": ["value"]
                }
              ],
              "additionalProperties": false
            }
          ]
        }
      },
      "additionalProperties": false
    },
    "local_only": {
      "type": "boolean",
      "description": "If true, only use this directory's config (don't merge with parent configs)",
      "default": false
    },
    "ignore_global": {
      "type": "boolean",
      "description": "If true, ignore global config (start fresh from this directory)",
      "default": false
    }
  },
  "additionalProperties": false
}`
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

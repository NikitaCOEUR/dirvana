//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/invopop/jsonschema"
)

// SchemaConfig represents the root configuration for schema generation
type SchemaConfig struct {
	Aliases      map[string]AliasValue `json:"aliases,omitempty" jsonschema:"description=Shell aliases - shortcuts for common commands"`
	Functions    map[string]string     `json:"functions,omitempty" jsonschema:"description=Shell functions - reusable command sequences"`
	Env          map[string]EnvValue   `json:"env,omitempty" jsonschema:"description=Environment variables (static or dynamic via shell commands)"`
	LocalOnly    bool                  `json:"local_only,omitempty" jsonschema:"description=If true only use this directory's config (don't merge with parent configs),default=false"`
	IgnoreGlobal bool                  `json:"ignore_global,omitempty" jsonschema:"description=If true ignore global config (start fresh from this directory),default=false"`
}

// AliasValue represents either a simple string command or a complex alias config
type AliasValue struct {
	Simple  string       `json:"-"`
	Complex *AliasConfig `json:"-"`
}

// AliasConfig represents an advanced alias with completion and conditions
type AliasConfig struct {
	Command    string           `json:"command" jsonschema:"required,minLength=1,description=Command to execute"`
	Completion *CompletionValue `json:"completion,omitempty" jsonschema:"description=Completion configuration (optional)"`
	When       *Condition       `json:"when,omitempty" jsonschema:"description=Conditions that must be met for the alias to execute"`
	Else       string           `json:"else,omitempty" jsonschema:"description=Fallback command to execute if conditions are not met"`
}

// CompletionValue can be string, bool, or CompletionConfig
type CompletionValue struct {
	Inherit string            `json:"-"`
	Disable bool              `json:"-"`
	Custom  *CompletionConfig `json:"-"`
}

// CompletionConfig for custom shell completion
type CompletionConfig struct {
	Bash string `json:"bash,omitempty" jsonschema:"description=Bash completion code"`
	Zsh  string `json:"zsh,omitempty" jsonschema:"description=Zsh completion code"`
}

// Condition represents conditions for alias execution
type Condition struct {
	File    string      `json:"file,omitempty" jsonschema:"description=Path to file that must exist (supports env var expansion like $VAR)"`
	Var     string      `json:"var,omitempty" jsonschema:"description=Environment variable that must be set and non-empty"`
	Dir     string      `json:"dir,omitempty" jsonschema:"description=Path to directory that must exist (supports env var expansion)"`
	Command string      `json:"command,omitempty" jsonschema:"description=Command that must exist in PATH"`
	All     []Condition `json:"all,omitempty" jsonschema:"minItems=1,description=All conditions must be true (AND logic)"`
	Any     []Condition `json:"any,omitempty" jsonschema:"minItems=1,description=At least one condition must be true (OR logic)"`
}

// EnvValue represents either a static string or dynamic shell command
type EnvValue struct {
	Static  string     `json:"-"`
	Dynamic *EnvConfig `json:"-"`
}

// EnvConfig for dynamic environment variables
type EnvConfig struct {
	Sh    string `json:"sh,omitempty" jsonschema:"minLength=1,description=Shell command to execute (output becomes env var value)"`
	Value string `json:"value,omitempty" jsonschema:"description=Alternative: static value (use string directly instead)"`
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

// JSONSchema implements custom schema generation for AliasValue
func (AliasValue) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type:        "string",
				MinLength:   uint64Ptr(1),
				Description: "Simple alias: command to execute (auto-detects completion)",
			},
			{
				Ref: "#/$defs/AliasConfig",
			},
		},
	}
}

// JSONSchema implements custom schema generation for CompletionValue
func (CompletionValue) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type:        "string",
				MinLength:   uint64Ptr(1),
				Description: "Inherit completion from another command (e.g., 'git')",
			},
			{
				Type:        "boolean",
				Description: "Set to false to disable completion",
			},
			{
				Ref: "#/$defs/CompletionConfig",
			},
		},
	}
}

// JSONSchema implements custom schema generation for EnvValue
func (EnvValue) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Type:        "string",
				Description: "Static environment variable value",
			},
			{
				Ref: "#/$defs/EnvConfig",
			},
		},
	}
}

func main() {
	r := &jsonschema.Reflector{
		DoNotReference:             false,
		ExpandedStruct:             false,
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
	}

	schema := r.Reflect(&SchemaConfig{})

	// Add missing definitions that are referenced via $ref
	// Extract just the type definition from each reflected schema
	aliasConfigSchema := r.ReflectFromType(reflect.TypeOf(AliasConfig{}))
	completionConfigSchema := r.ReflectFromType(reflect.TypeOf(CompletionConfig{}))
	conditionSchema := r.ReflectFromType(reflect.TypeOf(Condition{}))
	envConfigSchema := r.ReflectFromType(reflect.TypeOf(EnvConfig{}))

	// Get the actual definition from each schema's $defs
	if def, ok := aliasConfigSchema.Definitions["AliasConfig"]; ok {
		schema.Definitions["AliasConfig"] = def
		// Also add nested defs (CompletionValue, Condition)
		for k, v := range aliasConfigSchema.Definitions {
			if k != "AliasConfig" {
				schema.Definitions[k] = v
			}
		}
	}
	if def, ok := completionConfigSchema.Definitions["CompletionConfig"]; ok {
		schema.Definitions["CompletionConfig"] = def
	}
	if def, ok := conditionSchema.Definitions["Condition"]; ok {
		schema.Definitions["Condition"] = def
	}
	if def, ok := envConfigSchema.Definitions["EnvConfig"]; ok {
		schema.Definitions["EnvConfig"] = def
	}

	// Customize SchemaConfig to use patternProperties for aliases, functions, and env
	if schemaConfig, ok := schema.Definitions["SchemaConfig"]; ok {
		aliasPattern := "^[a-zA-Z_][a-zA-Z0-9_-]*$"
		funcPattern := "^[a-zA-Z_][a-zA-Z0-9_-]*$"
		envPattern := "^[a-zA-Z_][a-zA-Z0-9_]*$"

		// Convert aliases from additionalProperties to patternProperties
		if aliases, ok := schemaConfig.Properties.Get("aliases"); ok {
			aliases.PatternProperties = map[string]*jsonschema.Schema{
				aliasPattern: aliases.AdditionalProperties,
			}
			aliases.AdditionalProperties = jsonschema.FalseSchema
		}

		// Convert functions from additionalProperties to patternProperties
		if functions, ok := schemaConfig.Properties.Get("functions"); ok {
			functions.PatternProperties = map[string]*jsonschema.Schema{
				funcPattern: functions.AdditionalProperties,
			}
			functions.AdditionalProperties = jsonschema.FalseSchema
		}

		// Convert env from additionalProperties to patternProperties
		if env, ok := schemaConfig.Properties.Get("env"); ok {
			env.PatternProperties = map[string]*jsonschema.Schema{
				envPattern: env.AdditionalProperties,
			}
			env.AdditionalProperties = jsonschema.FalseSchema
		}
	}

	// Use draft-07 for IDE compatibility
	schema.Version = "http://json-schema.org/draft-07/schema#"
	schema.ID = "https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json"
	schema.Title = "Dirvana Configuration"
	schema.Description = "Configuration file for Dirvana - Reach directory nirvana"

	// Generate JSON
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling schema: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	outputPath := "schema.json"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Schema generated: %s\n", outputPath)
}

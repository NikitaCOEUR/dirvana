package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NikitaCOEUR/dirvana/internal/config"
)

// Validate validates a Dirvana configuration file
func Validate(configPath string) error {
	// If no path provided, look for config in current directory
	if configPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try to find a config file
		for _, name := range config.SupportedConfigNames {
			path := filepath.Join(currentDir, name)
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return fmt.Errorf("no config file found in current directory")
		}
	}

	fmt.Printf("Validating: %s\n\n", configPath)

	// Read file content for schema validation
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// First validate with JSON Schema
	result, err := config.ValidateWithSchema(configPath, content)
	if err != nil {
		return err
	}

	// If schema validation passes, run additional custom validations
	if result.Valid {
		customResult, err := config.Validate(configPath)
		if err != nil {
			return err
		}
		// Merge results
		if !customResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, customResult.Errors...)
		}
	}

	if result.Valid {
		fmt.Println("✅ Configuration is valid!")
		return nil
	}

	// Display errors
	fmt.Println("❌ Configuration has errors:")
	for i, validationErr := range result.Errors {
		fmt.Printf("%d. [%s] %s\n", i+1, validationErr.Field, validationErr.Message)
	}

	fmt.Printf("\nFound %d error(s)\n", len(result.Errors))

	// Return non-zero exit code
	return fmt.Errorf("validation failed")
}

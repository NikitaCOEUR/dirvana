package cli

import (
	"fmt"
	"os"

	"github.com/NikitaCOEUR/dirvana/internal/config"
)

// Schema displays or exports the JSON Schema for Dirvana configuration files
func Schema(outputPath string) error {
	schemaJSON := config.GetSchemaJSON()

	// If output path is provided, write to file
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(schemaJSON), 0644); err != nil {
			return fmt.Errorf("failed to write schema to %s: %w", outputPath, err)
		}
		fmt.Printf("JSON Schema written to: %s\n", outputPath)
		return nil
	}

	// Otherwise, print to stdout
	fmt.Println(schemaJSON)
	return nil
}

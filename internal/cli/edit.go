package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/NikitaCOEUR/dirvana/internal/config"
)

// Edit opens the config file in the user's editor
func Edit() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Look for existing config file
	var configPath string
	for _, name := range config.SupportedConfigNames {
		path := filepath.Join(currentDir, name)
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	// If no config exists, create default one
	if configPath == "" {
		configPath = filepath.Join(currentDir, ".dirvana.yml")
		defaultContent := `# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json
# Dirvana configuration
# See: https://github.com/NikitaCOEUR/dirvana

# Shell aliases
aliases:
  # ll: ls -la

# Shell functions
functions:
  # greet: echo "Hello, $1"

# Environment variables
env:
  # PROJECT_ROOT: .
  # DYNAMIC_VAR:
  #   sh: date +%s

# Prevent merging with parent configs
# local_only: false

# Ignore global config
# ignore_global: false
`
		if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		fmt.Printf("Created new config: %s\n", configPath)
	} else {
		fmt.Printf("Opening config: %s\n", configPath)
	}

	// Get editor from environment or use defaults
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"nano", "vim", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}

	if editor == "" {
		return fmt.Errorf("no editor found. Set $EDITOR or $VISUAL environment variable")
	}

	// Open editor
	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/NikitaCOEUR/dirvana/internal/config"
)

// Edit opens the config file in the user's editor
func Edit(global bool) error {
	var configPath string

	if global {
		// Edit global config
		globalPath, err := config.GetGlobalConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get global config path: %w", err)
		}
		configPath = globalPath

		// If global config doesn't exist, create it
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Create directory if it doesn't exist
			configDir := filepath.Dir(configPath)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			// Note: Will be created with default content below
		}
	} else {
		// Edit local config
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Look for existing config file
		for _, name := range config.SupportedConfigNames {
			path := filepath.Join(currentDir, name)
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		// If no config exists, use default name
		if configPath == "" {
			configPath = filepath.Join(currentDir, ".dirvana.yml")
		}
	}

	// If config doesn't exist, create default one
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultContent := `# yaml-language-server: $schema=https://raw.githubusercontent.com/NikitaCOEUR/dirvana/main/schema/dirvana.schema.json
# Dirvana configuration file
# Documentation: https://github.com/NikitaCOEUR/dirvana

# Shell aliases
aliases:
  # Simple string aliases (auto-detects completion)
  # g: git

  # Advanced format with completion control
  # tf:
  #  command: task terraform --
  #  completion: terraform  # Inherits terraform's auto-completion

# Shell functions - reusable command sequences with parameters
functions:
  # Simple greeting function
  # greet: |
  #   echo "Hello, $1!"

# Environment variables
env:
  # Static values
  # PROJECT_NAME: myproject

  # Dynamic values from shell commands (evaluated on load)
  # CURRENT_USER:
  #	  sh: whoami

# Configuration flags
# Set to true to ignore parent configs (only use this directory's config)
# local_only: false

# Set to true to ignore global config (~/.config/dirvana/global.yml)
# ignore_global: false
`
		if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		if global {
			fmt.Printf("Created new global config: %s\n", configPath)
		} else {
			fmt.Printf("Created new config: %s\n", configPath)
		}
	} else {
		if global {
			fmt.Printf("Opening global config: %s\n", configPath)
		} else {
			fmt.Printf("Opening config: %s\n", configPath)
		}
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

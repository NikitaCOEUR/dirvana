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
		configPath = config.FindConfigInDir(currentDir)

		// If no config exists, use default name
		if configPath == "" {
			configPath = filepath.Join(currentDir, ".dirvana.yml")
		}
	}

	// If config doesn't exist, create default one
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(sampleConfig), 0644); err != nil {
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

	// Get editor from environment (POSIX convention: VISUAL wins over EDITOR)
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
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

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/internal/derrors"
)

// Init creates a sample .dirvana.yml config file in the current directory or global config
func Init(global bool) error {
	var configPath string

	if global {
		// Create global config
		globalPath, err := config.GetGlobalConfigPath()
		if err != nil {
			return derrors.NewConfigurationError("", "failed to get global config path", err)
		}
		configPath = globalPath

		// Create directory if it doesn't exist
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return derrors.NewConfigurationError(configPath, "failed to create config directory", err)
		}
	} else {
		// Create local config
		currentDir, err := os.Getwd()
		if err != nil {
			return derrors.NewExecutionError("init", "failed to get current directory", err)
		}
		configPath = filepath.Join(currentDir, ".dirvana.yml")
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return derrors.NewAlreadyExistsError(configPath, fmt.Sprintf("config file already exists: %s", configPath))
	}

	if err := os.WriteFile(configPath, []byte(sampleConfig), 0644); err != nil {
		return derrors.NewConfigurationError(configPath, "failed to create config file", err)
	}

	if global {
		fmt.Printf("Created global config: %s\n", configPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit the config file to suit your needs")
		fmt.Println("  2. Run 'dirvana edit --global' to edit the global config")
		fmt.Println("  3. The global config will be automatically loaded in all directories")
	} else {
		fmt.Printf("Created sample config: %s\n", configPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit the config file to suit your needs")
		fmt.Println("  2. Run 'dirvana allow' to authorize this directory")
		fmt.Println("  3. Run 'dirvana setup' to install the shell hook")
	}

	return nil
}

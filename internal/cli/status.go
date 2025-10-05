package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikitaCOEUR/dirvana/internal/config"
	"github.com/NikitaCOEUR/dirvana/pkg/version"
)

// StatusParams contains parameters for the Status command
type StatusParams struct {
	CachePath string
	AuthPath  string
}

// Status displays the current Dirvana configuration status
func Status(params StatusParams) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("📂 Current directory: %s\n\n", currentDir)

	comps, err := initializeComponents(params.CachePath, params.AuthPath)
	if err != nil {
		return err
	}

	allowed, err := displayAuthStatus(comps, currentDir)
	if err != nil {
		return err
	}

	configFiles, hasGlobal, err := displayConfigHierarchyWithAuth(comps, currentDir)
	if err != nil {
		return err
	}

	// If not authorized or no configs, stop here
	if !allowed || (len(configFiles) == 0 && !hasGlobal) {
		return nil
	}

	// Load and display configuration details
	return displayConfigDetails(comps, currentDir, configFiles)
}

func displayAuthStatus(comps *components, currentDir string) (bool, error) {
	allowed, err := comps.auth.IsAllowed(currentDir)
	if err != nil {
		return false, fmt.Errorf("failed to check authorization: %w", err)
	}

	if allowed {
		fmt.Println("🔒 Authorization: ✓ Authorized")
	} else {
		fmt.Println("🔒 Authorization: ✗ Not authorized")
		fmt.Printf("   Run 'dirvana allow %s' to authorize\n", currentDir)
	}
	fmt.Println()
	return allowed, nil
}

// displayConfigHierarchyWithAuth displays the configuration hierarchy with authorization status
func displayConfigHierarchyWithAuth(comps *components, currentDir string) ([]string, bool, error) {
	configFiles, err := config.FindConfigFiles(currentDir)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find config files: %w", err)
	}

	globalPath, err := config.GetGlobalConfigPath()
	hasGlobal := false
	if err == nil {
		if _, err := os.Stat(globalPath); err == nil {
			hasGlobal = true
		}
	}

	fmt.Println("📝 Configuration hierarchy:")
	
	// Display global config
	if hasGlobal {
		fmt.Printf("   1. %s (global) ✓\n", globalPath)
	}
	
	if len(configFiles) == 0 && !hasGlobal {
		fmt.Println("   No configuration files found")
	} else {
		offset := 1
		if hasGlobal {
			offset = 2
		}
		
		// Display each config with authorization status
		for i, path := range configFiles {
			// Extract directory from config file path
			configDir := filepath.Dir(path)
			
			// Check if this directory is authorized
			allowed, err := comps.auth.IsAllowed(configDir)
			status := "✓"
			statusText := ""
			if err != nil || !allowed {
				status = "✗"
				statusText = " (not authorized)"
			}
			
			fmt.Printf("   %d. %s %s%s\n", i+offset, path, status, statusText)
		}
	}
	fmt.Println()

	return configFiles, hasGlobal, nil
}

func displayConfigDetails(comps *components, currentDir string, configFiles []string) error {
	merged, _, err := comps.config.LoadHierarchyWithAuth(currentDir, comps.auth)
	if err != nil {
		return fmt.Errorf("failed to load config hierarchy: %w", err)
	}

	displayAliases(merged.Aliases)
	displayFunctions(merged.Functions)
	displayEnvVars(merged)
	displayCacheStatus(comps, currentDir, configFiles)
	displayFlags(merged)

	return nil
}

func displayAliases(aliases map[string]interface{}) {
	fmt.Println("🔗 Aliases:")
	if len(aliases) == 0 {
		fmt.Println("   None")
	} else {
		// Parse aliases properly
		for name, value := range aliases {
			var cmd string
			switch v := value.(type) {
			case string:
				cmd = v
			case map[string]interface{}:
				if c, ok := v["command"].(string); ok {
					cmd = c
				}
			}
			if cmd != "" {
				fmt.Printf("   %s → %s\n", name, cmd)
			}
		}
	}
	fmt.Println()
}

func displayFunctions(functions map[string]string) {
	fmt.Println("⚙️  Functions:")
	if len(functions) == 0 {
		fmt.Println("   None")
	} else {
		for name := range functions {
			fmt.Printf("   %s()\n", name)
		}
	}
	fmt.Println()
}

func displayEnvVars(merged *config.Config) {
	staticEnv, shellEnv := merged.GetEnvVars()
	fmt.Println("🌍 Environment variables:")

	if len(staticEnv) == 0 && len(shellEnv) == 0 {
		fmt.Println("   None")
	} else {
		if len(staticEnv) > 0 {
			fmt.Println("   Static:")
			for name, value := range staticEnv {
				displayValue := truncateString(value, 50)
				fmt.Printf("      %s=%s\n", name, displayValue)
			}
		}
		if len(shellEnv) > 0 {
			fmt.Println("   Dynamic (shell):")
			for name, cmd := range shellEnv {
				displayCmd := truncateString(cmd, 50)
				fmt.Printf("      %s=$(%s)\n", name, displayCmd)
			}
		}
	}
	fmt.Println()
}

func displayCacheStatus(comps *components, currentDir string, configFiles []string) {
	configHash := ""
	if len(configFiles) > 0 {
		mainConfig := configFiles[len(configFiles)-1]
		configHash, _ = comps.config.Hash(mainConfig)
	}

	cacheValid := comps.cache.IsValid(currentDir, configHash, version.Version)
	fmt.Println("💾 Cache status:")
	if cacheValid {
		entry, _ := comps.cache.Get(currentDir)
		fmt.Println("   ✓ Cache hit")
		fmt.Printf("   Updated: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
		if entry.LocalOnly {
			fmt.Println("   Local only: yes")
		}
	} else {
		fmt.Println("   ✗ Cache miss (will be regenerated on next cd)")
	}
	fmt.Println()
}

func displayFlags(merged *config.Config) {
	fmt.Println("🏴 Flags:")
	flags := []string{}
	if merged.LocalOnly {
		flags = append(flags, "local_only")
	}
	if merged.IgnoreGlobal {
		flags = append(flags, "ignore_global")
	}
	if len(flags) == 0 {
		fmt.Println("   None")
	} else {
		fmt.Printf("   %s\n", strings.Join(flags, ", "))
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

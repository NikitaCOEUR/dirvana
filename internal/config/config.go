// Package config handles loading and parsing of Dirvana configuration files.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// SupportedConfigNames contains supported configuration file names (in order of preference)
var SupportedConfigNames = []string{
	".dirvana.yml",
	".dirvana.yaml",
	".dirvana.toml",
	".dirvana.json",
}

const (
	// GlobalConfigName is the name of the global config file
	GlobalConfigName = "global.yml"
)

// EnvVar represents an environment variable that can be static or dynamic (shell command)
type EnvVar struct {
	Value string // Static value or result of shell command
	Sh    string // Shell command to execute (mutually exclusive with Value)
}

// CompletionConfig represents shell completion configuration for an alias
type CompletionConfig struct {
	Bash string `koanf:"bash"` // Bash completion code
	Zsh  string `koanf:"zsh"`  // Zsh completion code
}

// AliasConfig represents an alias with optional completion override
type AliasConfig struct {
	Command    string      // The command to execute
	Completion interface{} // Can be: string (inherit), false (disable), or CompletionConfig object
}

// Config represents a dirvana configuration
type Config struct {
	Aliases      map[string]interface{} `koanf:"aliases"` // Can be string or AliasConfig struct
	Functions    map[string]string      `koanf:"functions"`
	Env          map[string]interface{} `koanf:"env"` // Can be string or EnvVar struct
	LocalOnly    bool                   `koanf:"local_only"`
	IgnoreGlobal bool                   `koanf:"ignore_global"`
}

// GetAliases returns a normalized map of alias name to AliasConfig
func (c *Config) GetAliases() map[string]AliasConfig {
	result := make(map[string]AliasConfig)

	for name, value := range c.Aliases {
		switch v := value.(type) {
		case string:
			// Simple string format: "alias: command"
			result[name] = AliasConfig{
				Command:    v,
				Completion: nil, // Auto-detect
			}
		case map[string]interface{}:
			// Object format with command and optional completion
			alias := AliasConfig{}

			if cmd, ok := v["command"].(string); ok {
				alias.Command = cmd
			}

			if comp, exists := v["completion"]; exists {
				switch c := comp.(type) {
				case string:
					// Inherit from command: "completion: git"
					alias.Completion = c
				case bool:
					// Disable: "completion: false"
					if !c {
						alias.Completion = false
					}
				case map[string]interface{}:
					// Custom completion with bash/zsh
					compCfg := CompletionConfig{}
					if bash, ok := c["bash"].(string); ok {
						compCfg.Bash = bash
					}
					if zsh, ok := c["zsh"].(string); ok {
						compCfg.Zsh = zsh
					}
					alias.Completion = compCfg
				}
			}

			result[name] = alias
		}
	}

	return result
}

// GetEnvVars returns a map of environment variable names to their resolved values or shell commands
func (c *Config) GetEnvVars() (map[string]string, map[string]string) {
	staticVars := make(map[string]string)
	shellVars := make(map[string]string)

	for key, value := range c.Env {
		switch v := value.(type) {
		case string:
			// Simple string value
			staticVars[key] = v
		case map[string]interface{}:
			// Structured EnvVar with 'sh' field
			if sh, ok := v["sh"].(string); ok && sh != "" {
				shellVars[key] = sh
			} else if val, ok := v["value"].(string); ok {
				staticVars[key] = val
			}
		}
	}

	return staticVars, shellVars
}

// Loader handles loading and parsing configuration files
type Loader struct {
	k *koanf.Koanf
}

// New creates a new config loader
func New() *Loader {
	return &Loader{
		k: koanf.New("."),
	}
}

// Load reads and parses a configuration file
func (l *Loader) Load(path string) (*Config, error) {
	// Create a new koanf instance for isolated loading
	k := koanf.New(".")

	// Determine parser based on file extension
	ext := strings.ToLower(filepath.Ext(path))
	var parser koanf.Parser

	switch ext {
	case ".yml", ".yaml":
		parser = yaml.Parser()
	case ".toml":
		parser = toml.Parser()
	case ".json":
		parser = json.Parser()
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	// Load the file
	if err := k.Load(file.Provider(path), parser); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Unmarshal into Config struct
	cfg := &Config{
		Aliases:   make(map[string]interface{}),
		Functions: make(map[string]string),
		Env:       make(map[string]interface{}),
	}

	if err := k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// Hash computes SHA-256 hash of a config file
func (l *Loader) Hash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// Merge merges parent and child configs, with child taking precedence
// If child has LocalOnly=true, parent is ignored
func Merge(parent, child *Config) *Config {
	if child.LocalOnly {
		return child
	}

	merged := &Config{
		Aliases:      make(map[string]interface{}),
		Functions:    make(map[string]string),
		Env:          make(map[string]interface{}),
		LocalOnly:    child.LocalOnly,
		IgnoreGlobal: child.IgnoreGlobal,
	}

	// Merge aliases (parent first, child overrides)
	for k, v := range parent.Aliases {
		merged.Aliases[k] = v
	}
	for k, v := range child.Aliases {
		merged.Aliases[k] = v
	}

	// Merge functions
	for k, v := range parent.Functions {
		merged.Functions[k] = v
	}
	for k, v := range child.Functions {
		merged.Functions[k] = v
	}

	// Merge env vars (interface{} type)
	for k, v := range parent.Env {
		merged.Env[k] = v
	}
	for k, v := range child.Env {
		merged.Env[k] = v
	}

	return merged
}

// GetGlobalConfigPath returns the path to the global config file
func GetGlobalConfigPath() (string, error) {
	// Try XDG_CONFIG_HOME first
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		// Fallback to ~/.config
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, "dirvana", GlobalConfigName), nil
}

// FindConfigFiles searches for config files from current dir up to root
// Returns paths in order from root to leaf (for proper merging)
func FindConfigFiles(startDir string) ([]string, error) {
	var configs []string
	currentDir := startDir

	// Walk up directory tree
	for {
		// Check for config files in current directory
		for _, name := range SupportedConfigNames {
			path := filepath.Join(currentDir, name)
			if _, err := os.Stat(path); err == nil {
				configs = append(configs, path)
				break // Only one config per directory
			}
		}

		// Move up to parent directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached root
			break
		}
		currentDir = parent
	}

	// Reverse to get root-to-leaf order
	for i, j := 0, len(configs)-1; i < j; i, j = i+1, j-1 {
		configs[i], configs[j] = configs[j], configs[i]
	}

	return configs, nil
}

// LoadHierarchy loads and merges all configs from global to current directory
// Order: global config → root → ... → parent → current
func (l *Loader) LoadHierarchy(dir string) (*Config, []string, error) {
	var allConfigFiles []string
	var merged *Config

	// Try to load global config first
	globalPath, err := GetGlobalConfigPath()
	if err == nil {
		if _, err := os.Stat(globalPath); err == nil {
			globalCfg, err := l.Load(globalPath)
			if err == nil {
				// Successfully loaded global config
				merged = globalCfg
				allConfigFiles = append(allConfigFiles, globalPath)
			}
			// If global config is invalid, just skip it - user can still use local configs
		}
	}

	// Find local config files (from root to current directory)
	configFiles, err := FindConfigFiles(dir)
	if err != nil {
		return nil, nil, err
	}

	// If no local configs and no global config, return empty config
	if len(configFiles) == 0 && merged == nil {
		return &Config{
			Aliases:   make(map[string]interface{}),
			Functions: make(map[string]string),
			Env:       make(map[string]interface{}),
		}, nil, nil
	}

	// Merge local configs
	for _, path := range configFiles {
		cfg, err := l.Load(path)
		if err != nil {
			return nil, append(allConfigFiles, configFiles...), err
		}

		// Check if this config wants to ignore global
		if cfg.IgnoreGlobal && merged != nil {
			// If first local config has ignore_global, start fresh
			if len(allConfigFiles) == 1 {
				merged = nil
				allConfigFiles = nil
			}
		}

		if merged == nil {
			merged = cfg
		} else {
			merged = Merge(merged, cfg)
		}

		allConfigFiles = append(allConfigFiles, path)

		// If local_only is set, stop merging
		if cfg.LocalOnly {
			break
		}
	}

	return merged, allConfigFiles, nil
}

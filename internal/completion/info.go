package completion

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DetectionInfo contains information about the detection cache
type DetectionInfo struct {
	Path     string
	Size     int64
	Commands map[string]string // command -> source type
}

// RegistryInfo contains information about the completion registry
type RegistryInfo struct {
	Path       string
	Size       int64
	ToolsCount int
}

// ScriptInfo contains information about a downloaded completion script
type ScriptInfo struct {
	Tool string
	Path string
	Size int64
}

// GetDetectionCacheInfo returns information about the detection cache
func GetDetectionCacheInfo(cacheDir string) (*DetectionInfo, error) {
	detectionCachePath := filepath.Join(cacheDir, "completion-detection.json")

	info, err := os.Stat(detectionCachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache yet
		}
		return nil, err
	}

	result := &DetectionInfo{
		Path:     detectionCachePath,
		Size:     info.Size(),
		Commands: make(map[string]string),
	}

	// Load detected commands
	data, err := os.ReadFile(detectionCachePath)
	if err != nil {
		return result, nil // Return partial info
	}

	type cacheEntry struct {
		CompleterType string `json:"completer_type"`
	}
	var detections map[string]cacheEntry
	if err := json.Unmarshal(data, &detections); err != nil {
		return result, nil // Return partial info
	}

	for cmd, entry := range detections {
		result.Commands[cmd] = entry.CompleterType
	}

	return result, nil
}

// GetRegistryInfo returns information about the completion registry
func GetRegistryInfo(cacheDir string) (*RegistryInfo, error) {
	registryPath := filepath.Join(cacheDir, "completion-registry-v1.yml")

	info, err := os.Stat(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No registry yet
		}
		return nil, err
	}

	result := &RegistryInfo{
		Path: registryPath,
		Size: info.Size(),
	}

	// Load registry to count tools
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return result, nil // Return partial info
	}

	type registryConfig struct {
		Tools map[string]interface{} `yaml:"tools"`
	}
	var reg registryConfig
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return result, nil // Return partial info
	}

	result.ToolsCount = len(reg.Tools)

	return result, nil
}

// GetDownloadedScripts returns information about downloaded completion scripts
func GetDownloadedScripts(cacheDir string) ([]ScriptInfo, error) {
	scriptsPath := filepath.Join(cacheDir, "completion-scripts")
	bashScriptsPath := filepath.Join(scriptsPath, "bash")

	entries, err := os.ReadDir(bashScriptsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No scripts yet
		}
		return nil, err
	}

	var scripts []ScriptInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		scriptPath := filepath.Join(bashScriptsPath, entry.Name())
		info, err := os.Stat(scriptPath)
		if err != nil {
			continue
		}

		scripts = append(scripts, ScriptInfo{
			Tool: entry.Name(),
			Path: scriptPath,
			Size: info.Size(),
		})
	}

	return scripts, nil
}

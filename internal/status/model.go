package status

import (
	"time"

	"github.com/NikitaCOEUR/dirvana/internal/config"
)

// Data contains all the information to display in status
type Data struct {
	// Header
	CurrentDir string
	Version    string

	// System & Installation
	Shell         string
	HookInstalled bool
	RCFile        string
	CachePath     string
	AuthPath      string

	// Authorization
	Authorized     bool
	HasAnyConfig   bool // Whether there's any config (local or global) to authorize

	// Configuration
	GlobalConfig *config.GlobalInfo
	LocalConfigs []config.FileInfo

	// Config Details
	Aliases   map[string]config.AliasInfo
	Functions []string
	EnvStatic map[string]string
	EnvShell  map[string]config.EnvShellInfo
	Flags     []string

	// Cache
	CacheFileSize     int64
	CacheTotalEntries int
	CacheValid        bool
	CacheUpdated      time.Time
	CacheLocalOnly    bool

	// Completion
	CompletionDetection *CompletionDetectionInfo
	CompletionRegistry  *CompletionRegistryInfo
	CompletionScripts   []CompletionScriptInfo
	CompletionOverrides map[string]string // alias -> command
}

// CompletionDetectionInfo contains detection cache information
type CompletionDetectionInfo struct {
	Path     string
	Size     int64
	Commands map[string]string // command -> source type
}

// CompletionRegistryInfo contains registry information
type CompletionRegistryInfo struct {
	Path       string
	Size       int64
	ToolsCount int
}

// CompletionScriptInfo contains information about a downloaded script
type CompletionScriptInfo struct {
	Tool string
	Path string
	Size int64
}


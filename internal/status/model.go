package status

import (
	"time"
)

// Data contains all the information to display in status
type Data struct {
	// Header
	CurrentDir string
	Version    string

	// System & Installation. Shell is empty when it could not be identified,
	// which is a different thing from an unsupported one.
	Shell         string
	HookInstalled bool
	// HookOutdated marks a hook installed by an older release: it still runs,
	// but its code no longer matches what this binary generates. Which dirvana
	// it points at is not part of that judgement - see HookBroken.
	HookOutdated bool
	// HookBroken means the hook points at a dirvana that is gone, so it fails
	// silently on every cd.
	HookBroken bool
	// HookBinary is the dirvana the installed hook invokes.
	HookBinary string
	// HookFile holds the hook code; RCFile is the shell config pulling it in,
	// and the two differ for every strategy but the inline one.
	HookFile  string
	RCFile    string
	CachePath string
	AuthPath  string

	// Authorization
	Authorized   bool
	HasAnyConfig bool // Whether there's any config (local or global) to authorize

	// Configuration
	GlobalConfig *GlobalInfo
	LocalConfigs []FileInfo

	// Config Details
	Aliases   map[string]AliasInfo
	Functions []string
	EnvStatic map[string]string
	EnvShell  map[string]EnvShellInfo
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

// Package auth provides authorization management for Dirvana projects.
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const currentAuthVersion = 2

// GetAuth returns the DirAuth structure for a given directory path
func (a *Auth) GetAuth(path string) *DirAuth {
	a.mu.RLock()
	defer a.mu.RUnlock()
	normalized := normalizePath(path)
	return a.authorized[normalized]
}

// RequiresShellApproval returns true if shell command approval is needed for the directory
func (a *Auth) RequiresShellApproval(dir string, shellCmds map[string]string) bool {
	if len(shellCmds) == 0 {
		return false
	}
	auth := a.GetAuth(dir)
	if auth == nil || !auth.Allowed {
		return false // Directory authorization required first
	}
	currentHash := hashShellCommands(shellCmds)
	return auth.ShellCommandsHash == "" || auth.ShellCommandsHash != currentHash
}

// hashShellCommands computes a deterministic hash of shell commands
func hashShellCommands(cmds map[string]string) string {
	keys := make([]string, 0, len(cmds))
	for k := range cmds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		// Write to hash (error can be safely ignored as hash.Hash.Write never fails)
		_, _ = fmt.Fprintf(h, "%s=%s\n", k, cmds[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ApproveShellCommands saves shell command approval for a directory
func (a *Auth) ApproveShellCommands(dir string, shellCmds map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	normalized := normalizePath(dir)
	auth := a.authorized[normalized]
	if auth == nil {
		return fmt.Errorf("directory not authorized")
	}
	auth.ShellCommandsHash = hashShellCommands(shellCmds)
	auth.ShellApprovedAt = time.Now()
	return a.persist()
}

// DirAuth stores the authorization state of a directory, including dynamic shell command approval
type DirAuth struct {
	Allowed           bool      `json:"allowed"`
	AllowedAt         time.Time `json:"allowed_at,omitempty"`
	ShellCommandsHash string    `json:"shell_commands_hash,omitempty"`
	ShellApprovedAt   time.Time `json:"shell_approved_at,omitempty"`
}

// File represents the v2 auth file structure with version metadata
type File struct {
	Version     int                 `json:"_version"`
	Directories map[string]*DirAuth `json:"directories"`
}

// Auth manages project directory authorization and shell command approval
type Auth struct {
	pathV1     string // V1 file (read-only, never modified)
	pathV2     string // V2 file (read/write)
	mu         sync.RWMutex
	authorized map[string]*DirAuth
}

// New creates or loads an Auth instance
func New(path string) (*Auth, error) {
	// path is the V1 path (e.g., authorized.json)
	// V2 path is derived by adding _v2 suffix
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	nameWithoutExt := base[:len(base)-len(ext)]
	pathV2 := filepath.Join(dir, nameWithoutExt+"_v2"+ext)

	a := &Auth{
		pathV1:     path,
		pathV2:     pathV2,
		authorized: make(map[string]*DirAuth),
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Try to load: V2 first (if exists), then V1 (read-only)
	if err := a.load(); err != nil && !os.IsNotExist(err) {
		// Start with empty state on errors
		a.authorized = make(map[string]*DirAuth)
	}

	return a, nil
}

// GetAuth returns the DirAuth structure for a given directory path

// Allow adds a directory to the authorized list
func (a *Auth) Allow(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	normalized := normalizePath(path)
	now := time.Now()
	if a.authorized[normalized] == nil {
		a.authorized[normalized] = &DirAuth{
			Allowed:   true,
			AllowedAt: now,
		}
	} else {
		a.authorized[normalized].Allowed = true
		a.authorized[normalized].AllowedAt = now
	}
	return a.persist()
}

// IsAllowed checks if a directory is authorized
func (a *Auth) IsAllowed(path string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	normalized := normalizePath(path)
	auth := a.authorized[normalized]
	return auth != nil && auth.Allowed, nil
}

// Revoke removes a directory from the authorized list
func (a *Auth) Revoke(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	normalized := normalizePath(path)
	delete(a.authorized, normalized)
	return a.persist()
}

// List returns all authorized directories
func (a *Auth) List() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	paths := make([]string, 0, len(a.authorized))
	for path := range a.authorized {
		paths = append(paths, path)
	}
	return paths
}

// Clear removes all authorized directories
func (a *Auth) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.authorized = make(map[string]*DirAuth)
	return a.persist()
}

// load reads authorized directories from disk
func (a *Auth) load() error {
	// Try V2 first
	if dataV2, err := os.ReadFile(a.pathV2); err == nil {
		return a.loadV2(dataV2)
	}

	// Fallback to V1 (read-only, never modified)
	dataV1, err := os.ReadFile(a.pathV1)
	if err != nil {
		return err
	}

	return a.loadV1(dataV1)
}

// loadV2 parses V2 format with version field
func (a *Auth) loadV2(data []byte) error {
	var authFile File
	if err := json.Unmarshal(data, &authFile); err != nil {
		return fmt.Errorf("invalid v2 auth file: %w", err)
	}

	if authFile.Version != 2 {
		return fmt.Errorf("unsupported auth file version: %d", authFile.Version)
	}

	a.authorized = make(map[string]*DirAuth)
	for path, auth := range authFile.Directories {
		if auth != nil {
			a.authorized[normalizePath(path)] = auth
		}
	}

	return nil
}

// loadV1 parses V1 format ([]string) - read-only, never writes back
func (a *Auth) loadV1(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}

	// V1 format: []string (array of authorized paths)
	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil {
		return fmt.Errorf("invalid v1 auth file: %w", err)
	}

	now := time.Now()
	a.authorized = make(map[string]*DirAuth)
	for _, path := range paths {
		a.authorized[normalizePath(path)] = &DirAuth{
			Allowed:   true,
			AllowedAt: now,
		}
	}

	// Don't auto-migrate - V1 stays untouched
	return nil
}

// persist writes authorized directories to disk in V2 format
func (a *Auth) persist() error {
	authFile := File{
		Version:     currentAuthVersion,
		Directories: a.authorized,
	}

	data, err := json.MarshalIndent(authFile, "", "  ")
	if err != nil {
		return err
	}
	// Always write to V2 file, never modify V1
	return os.WriteFile(a.pathV2, data, 0600)
}

// normalizePath removes trailing slashes and cleans the path
func normalizePath(path string) string {
	cleaned := filepath.Clean(path)
	return strings.TrimSuffix(cleaned, "/")
}

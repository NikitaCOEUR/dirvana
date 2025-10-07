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
		fmt.Fprintf(h, "%s=%s\n", k, cmds[k])
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

// Auth manages project directory authorization and shell command approval
type Auth struct {
	path       string
	mu         sync.RWMutex
	authorized map[string]*DirAuth
}

// New creates a new auth manager
func New(path string) (*Auth, error) {
	a := &Auth{
		path:       path,
		authorized: make(map[string]*DirAuth),
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Load existing authorized paths if file exists
	if err := a.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return a, nil
}

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
	data, err := os.ReadFile(a.path)
	if err != nil {
		return err
	}

	var auths map[string]*DirAuth
	if err := json.Unmarshal(data, &auths); err != nil {
		return err
	}

	a.authorized = make(map[string]*DirAuth)
	for path, auth := range auths {
		a.authorized[normalizePath(path)] = auth
	}

	return nil
}

// persist writes authorized directories to disk
func (a *Auth) persist() error {
	data, err := json.MarshalIndent(a.authorized, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.path, data, 0600)
}

// normalizePath removes trailing slashes and cleans the path
func normalizePath(path string) string {
	cleaned := filepath.Clean(path)
	return strings.TrimSuffix(cleaned, "/")
}
